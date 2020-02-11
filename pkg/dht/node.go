package dht

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/ksonbol/edgekv/utils"
)

// IDGenerator defines the type for generation IDs
type IDGenerator func(s string) string

// // IDBits is the number of bits contained in ID hashes
// const IDBits = 160 // sha1

// // IDChars is the number of characters contained in ID hashes in hex encoding
// const IDChars int = IDBits / 4

// Config specifies configurations for a dht node
type Config struct {
	IDBits  int
	IDChars int
	IDFunc  IDGenerator
}

var defaultConfig *Config = &Config{IDBits: 160, IDChars: 40, IDFunc: GenID}

// var defaultConfig *Config = &Config{IDBits: 5, IDChars: 3, idFunc: shortID}

// Node represents a node in the DHT overlay
type Node struct {
	Addr       string
	ID         string
	ft         []fingerEntry
	Transport  *transport
	pred       *Node
	server     *Server
	shutdownCh chan struct{}
	nodeJoinCh chan struct{}
	predMut    sync.RWMutex
	succMut    sync.RWMutex
	closeOnce  sync.Once
	Conf       *Config
	storage    Storage
	leaving    bool // set true when node is leaving the ring
}

// newBasicNode creates a node with the essential parameters
func newBasicNode(addr string, id string, conf *Config) *Node {
	return &Node{
		Addr:       addr,
		ID:         id,
		Conf:       conf,
		shutdownCh: make(chan struct{}),
		nodeJoinCh: make(chan struct{}),
	}
}

// NewLocalNode creates a new DHT node, initializing ID and transport
func NewLocalNode(addr string, st Storage, conf *Config) *Node {
	if conf == nil {
		conf = defaultConfig
	}
	n := newBasicNode(addr, conf.IDFunc(addr), conf)
	n.storage = st
	n.ft = initFT(n)
	fillFTFirstNode(n)
	n.SetPredecessor(n)
	hostname, port := utils.SplitAddress(addr)
	n.server = NewServer(hostname, port, n)
	n.server.RunInsecure()
	fmt.Printf("node %s: Server is running", addr)
	n.Transport = newTransport(n, nil, nil)
	return n
}

// NewRemoteNode returns a Node instance without the FT
func NewRemoteNode(addr string, id string, localTransport *transport, conf *Config) *Node {
	if conf == nil {
		conf = defaultConfig
	}
	if id == "" {
		id = conf.IDFunc(addr)
	}
	if conf == nil {
		conf = defaultConfig
	}
	n := newBasicNode(addr, id, conf)
	n.Conf = conf
	if localTransport == nil {
		// this is useful if we want to connect to a single node only
		n.Transport = newTransport(n, nil, nil)
	} else {
		// this is useful if we will connect to multiple nodes
		n.Transport = newTransport(n, localTransport.remotes, localTransport.mux)
	}
	return n
}

// Join node to a DHT with the help of helperNode
func (n *Node) Join(helperNode *Node) error {
	rand.Seed(time.Now().UnixNano())
	if helperNode == nil {
		// TODO: check if set predecessor is needed?
		// n.SetPredecessor(n)
	} else {
		succ, err := helperNode.FindSuccessorRPC(n.ft[0].start)
		if err != nil {
			return err
		}
		log.Printf("Setting node %s successor 1st time: %s\n", n.ID, succ.ID)

		n.SetSuccessor(succ)
	}
	go n.stabilize()
	go n.fixFingers()
	go n.checkPredecessor()
	time.Sleep(3 * time.Second) // wait for stabilization
	n.getFirstSnapshot()
	return nil
}

// GetSuccessorRPC returns the successor of n using an RPC call
func (n *Node) GetSuccessorRPC() (*Node, error) {
	return n.Transport.getSuccessor()
}

// GetPredecessorRPC returns the predecessor of n using an RPC call
func (n *Node) GetPredecessorRPC() (*Node, error) {
	return n.Transport.getPredecessor()
}

// FindSuccessorRPC finds the successor of a key an RPC call to n
// this may initiate more RPC calls
func (n *Node) FindSuccessorRPC(id string) (*Node, error) {
	return n.Transport.findSuccessor(id)
}

// ClosestPrecedingFingerRPC returns the closest node that precedes id
func (n *Node) ClosestPrecedingFingerRPC(id string) (*Node, error) {
	return n.Transport.closestPrecedingFinger(id)
}

// NotifyRPC notify node that n should be their predecessor
func (n *Node) NotifyRPC(node *Node) error {
	return n.Transport.notify(node)
}

// GetKVRPC gets the KV using the node stub
func (n *Node) GetKVRPC(key string) (string, error) {
	return n.Transport.getKV(key)
}

// PutKVRPC puts the KV using the node stub
func (n *Node) PutKVRPC(key, value string) error {
	return n.Transport.putKV(key, value)
}

// DelKVRPC deletes the KV using the node stub
func (n *Node) DelKVRPC(key string) error {
	return n.Transport.delKV(key)
}

// CanStoreRPC returns CanStore on the remote node
func (n *Node) CanStoreRPC(key string) (bool, error) {
	return n.Transport.canStore(key)
}

// RangeGetKVRPC remotely gets keys in range [startID, endID)
func (n *Node) RangeGetKVRPC(startID, endID string) (map[string]string, error) {
	return n.Transport.rangeGetKV(startID, endID)
}

// IsLeavingRPC checks if remote node is leaving the dht ring
func (n *Node) IsLeavingRPC() (bool, error) {
	return n.Transport.isLeaving()
}

// GetKV gets the KV from the connected edge group or from a remote group
func (n *Node) GetKV(key string) (string, error) {
	var err error
	if n.CanStore(key) {
		var val string
		if val, err = n.storage.GetKV(key); err != nil {
			return "", err
		}
		return val, nil
	}
	var succ *Node
	if succ, err = n.findSuccessor(key); err != nil {
		return "", err
	}
	return succ.GetKVRPC(key)
}

// PutKV puts the KV to the connected edge group or to a remote group
func (n *Node) PutKV(key, value string) error {
	var succ *Node
	var err error
	if n.CanStore(key) {
		return n.storage.PutKV(key, value)
	}
	if succ, err = n.findSuccessor(key); err != nil {
		return err
	}
	return succ.PutKVRPC(key, value)
}

// putKVLocal stores the kv-pair in the connected group if it is responsible for it
// otherwise, it deos nothing and returns an error (does not send to other remote groups)
func (n *Node) putKVLocal(key, value string) error {
	if n.CanStore(key) {
		return n.storage.PutKV(key, value)
	}
	return fmt.Errorf("Put request failed, key %s does not belong to this group", key)
}

// DelKV removes the KV from the connected edge group or from a remote group
func (n *Node) DelKV(key string) error {
	var succ *Node
	var err error
	if n.CanStore(key) {
		return n.storage.DelKV(key)
	}
	if succ, err = n.findSuccessor(key); err != nil {
		return err
	}
	return succ.DelKVRPC(key)
}

// RangeGetKV returns kv-pairs in specified range from connected edge group
func (n *Node) RangeGetKV(start, end string) (map[string]string, error) {
	return n.storage.RangeGetKV(start, end)
}

// Successor returns the successor of node n
func (n *Node) Successor() *Node {
	n.succMut.RLock()
	defer n.succMut.RUnlock()
	return n.ft[0].node
}

// Predecessor returns the predecessor of node n
func (n *Node) Predecessor() *Node {
	n.predMut.RLock()
	defer n.predMut.RUnlock()
	return n.pred
}

// SetPredecessor sets the predecessor of node n
func (n *Node) SetPredecessor(pred *Node) {
	n.predMut.Lock()
	n.pred = pred
	n.predMut.Unlock()
}

// SetSuccessor sets the successsor of node n in the ft
func (n *Node) SetSuccessor(succ *Node) {
	n.succMut.Lock()
	n.ft[0].node = succ
	n.succMut.Unlock()
}

// stabilize periodically checks if the successor is up-to-date and notifies them
// that this node should be their predecessor
func (n *Node) stabilize() {
	succ := n.Successor()
	var newSucc *Node
	var err error
	if n == succ {
		<-n.nodeJoinCh // if only node, wait until other nodes join
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			if n == succ {
				newSucc = succ.Predecessor()
			} else {
				leaving, er := succ.IsLeavingRPC()
				if er != nil {
					log.Fatalf("Failed to run the stabilizer algorithm %v", er)
				}
				if leaving {
					// if successor is leaving, use their successor
					newSucc, err = succ.GetSuccessorRPC()
					if err != nil {
						log.Fatalf("Failed to run the stabilizer algorithm %v", err)
					}
					n.SetSuccessor(newSucc)
					// we don't notify new successor here, they will realize us on their own
					continue
				}
				newSucc, err = succ.GetPredecessorRPC()
				if err != nil {
					log.Fatalf("Failed to run the stabilizer algorithm %v", err)
				}
			}
			if newSucc.ID != succ.ID {
				// if n == successor OR newSucc in (n, successor)
				if (n == succ) || (inInterval(newSucc.ID, incID(n.ID, n.Conf.IDChars), succ.ID)) {
					log.Printf("Replacing node %s old successor (%s) with %s\n",
						n.ID, succ.ID, newSucc.ID)
					n.SetSuccessor(newSucc)
				}
			}
			succ = n.Successor() // get the possibly updated successor
			if n != succ {
				succ.NotifyRPC(n)
			}
		case <-n.shutdownCh:
			ticker.Stop()
			return
		}
	}
}

// fixFingers periodically updates a finger chosen at random
func (n *Node) fixFingers() {
	if n == n.Successor() {
		<-n.nodeJoinCh // wait until other nodes join
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			if n == n.Successor() {
				continue // wait until the successor link is updated
			}
			i := rand.Intn(n.Conf.IDBits-1) + 1 // i in (1,IDBits)
			succ, err := n.findSuccessor(n.ft[i].start)
			if err != nil {
				log.Fatalf("Failed to get successor while running fixFingers %v", err)
			}
			n.ft[i].node = succ
		case <-n.shutdownCh:
			ticker.Stop()
			return
		}
	}
}

// fixFingers periodically checks if predecessor is available or leaving the network
func (n *Node) checkPredecessor() {
	var pred *Node
	var err error
	var leaving bool
	if n == n.Successor() {
		<-n.nodeJoinCh // wait until other nodes join
	}
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			pred = n.Predecessor()
			if n == pred {
				continue // wait until the predecessor link is updated
			}
			leaving, err = pred.IsLeavingRPC()
			if err != nil {
				log.Fatalf("Error while communicating with predecessor %v", err)
			}
			if leaving {
				pred, err = pred.GetPredecessorRPC()
				if err != nil {
					log.Fatalf("Error while communicating with predecessor %v", err)
				}
				n.SetPredecessor(pred)
			}
		case <-n.shutdownCh:
			ticker.Stop()
			return
		}
	}
}

func (n *Node) getFirstSnapshot() {
	succ := n.Successor()
	if n == succ {
		<-n.nodeJoinCh // if only node, wait until other nodes join
	}
	// get keys in range (predecessor, n.ID]
	kvs, err := succ.RangeGetKVRPC(incID(n.Predecessor().ID, n.Conf.IDChars), incID(n.ID, n.Conf.IDChars))
	if err != nil {
		log.Fatalf("Node initialization failed: could not copy keys from successor")
	}
	for k, v := range kvs {
		if err := n.putKVLocal(k, v); err != nil {
			log.Fatalf("Failed to add key to connected edge group %v", err)
		}
	}
}

// findSuccessor finds first node that follows ID
func (n *Node) findSuccessor(ID string) (*Node, error) {
	pred, err := n.findPredecessor(ID)
	if err != nil {
		return nil, err
	}
	var succ *Node
	if pred.ID == n.ID {
		succ = pred.Successor()
	} else {
		succ, err = pred.GetSuccessorRPC()
	}
	if err != nil {
		return nil, err
	}
	return succ, nil
}

// findPredecessor uses DHT RPCs to find the predecessor of a specific key (ID)
func (n *Node) findPredecessor(ID string) (*Node, error) {
	if n == n.Successor() {
		return n, nil // n is the only node in the network
	}
	next := n
	err := *new(error)
	succ := next.Successor() // current node, no need for RPC yet
	// while id not in (next.ID, next.Successor.ID]
	for !inInterval(ID, incID(next.ID, n.Conf.IDChars), incID(succ.ID, n.Conf.IDChars)) {
		if next == n {
			next = next.closestPrecedingFinger(ID)
		} else {
			next, err = next.ClosestPrecedingFingerRPC(ID)
			if err != nil {
				return nil, err
			}
		}
		// get next's successor for next iteration
		if n.ID == next.ID {
			succ = next.Successor()
		} else {
			succ, err = next.GetSuccessorRPC()
			if err != nil {
				return nil, err
			}
		}
	}
	return next, nil
}

func (n *Node) closestPrecedingFinger(ID string) *Node {
	for i := len(n.ft) - 1; i >= 0; i-- {
		fingerNode := n.ft[i].node
		if fingerNode == nil {
			fmt.Printf("FT[%d] with start=%s is not set\n", i, n.ft[i].start)
		}
		if inInterval(fingerNode.ID, incID(n.ID, n.Conf.IDChars), ID) {
			return fingerNode
		}
	}
	return n
}

// CanStore returns true if the key is the responisibility of this node and false otherwise
func (n *Node) CanStore(key string) bool {
	// keys is in (pred, n]
	return inInterval(key, incID(n.Predecessor().ID, n.Conf.IDChars), incID(n.ID, n.Conf.IDChars))
}

// GetFTID returns the ID with ft entry with index idx
func (n *Node) GetFTID(idx int) string {
	return n.ft[idx].node.ID
}

// Leave the DHT ring and stop the node
func (n *Node) Leave() error {
	n.setLeaving()
	close(n.shutdownCh)          // stop stabilize and fixFinger goroutines
	time.Sleep(60 * time.Second) // give enough time for other nodes to stabilize
	return n.shutDown()
	// TODO: any other tasks?
	// inform other nodes or copy keys to them before leaving?
}

func (n *Node) shutDown() error {
	if n.server != nil {
		n.server.stop()
	}
	for _, conn := range n.Transport.remotes {
		conn.Close()
	}
	n.Transport.shutdown()
	n.ft = nil
	n.pred = nil
	return nil
}

func (n *Node) isLeaving() bool {
	return n.leaving
}

func (n *Node) setLeaving() {
	n.leaving = true
}

// PrintFT prints the finger table of node n
func (n *Node) PrintFT() {
	for i, ent := range n.ft {
		fmt.Printf("%d %s %s\n", i, ent.start, ent.node.ID)
	}
}

// ZeroID return the Zero ID in the ring
func (n *Node) ZeroID() string {
	return appendZeros("0", n.Conf.IDChars)
}

// MaxID return the maximum possible ID in the ring
func (n *Node) MaxID() string {
	return appendZeros("f", n.Conf.IDChars)
}

// GenID creates a sha1 hash of a given string
func GenID(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	hash := hex.EncodeToString(h.Sum(nil))
	return hash
}
