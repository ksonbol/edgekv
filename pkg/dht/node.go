package dht

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/ksonbol/edgekv/utils"
)

// IDBits is the number of bits contained in ID hashes
const IDBits = 160 // sha1

// IDChars is the number of characters contained in ID hashes in hex encoding
const IDChars int = IDBits / 4

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
}

// newBasicNode creates a node with the essential parameters
func newBasicNode(addr string, id string) *Node {
	return &Node{Addr: addr, ID: id, shutdownCh: make(chan struct{}), nodeJoinCh: make(chan struct{})}
}

// NewLocalNode creates a new DHT node, initializing ID and transport
func NewLocalNode(addr string) *Node {
	n := newBasicNode(addr, genID(addr))
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
func NewRemoteNode(addr string, id string, localTransport *transport) *Node {
	if id == "" {
		id = genID(addr)
	}
	n := newBasicNode(addr, id)
	n.Transport = newTransport(n, localTransport.remotes, localTransport.mux)
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
		v1, _ := strconv.ParseInt(n.ID, 16, 32)
		v2, _ := strconv.ParseInt(succ.ID, 16, 32)
		log.Printf("Setting node %d successor 1st time: %d\n", v1,
			v2)

		n.SetSuccessor(succ)
	}
	go n.stabilize()
	go n.fixFingers()
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

// stabilize periodically checks the successor links
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
				newSucc, err = succ.GetPredecessorRPC()
				if err != nil {
					log.Fatalf("Failed to run the stabilizer algorithm %v", err)
				}
			}
			if newSucc.ID != succ.ID {
				// if n == successor OR newSucc in (n, successor)
				if (n == succ) || (inInterval(newSucc.ID, incID(n.ID), succ.ID)) {
					// log.Printf("Replacing node %s old successor (%s) with %s\n",
					// 	n.ID, succ.ID, newSucc.ID)
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
			i := rand.Intn(IDBits)
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
	for !inInterval(ID, incID(next.ID), incID(succ.ID)) {
		if next == n {
			next = next.closestPrecedingFinger(ID)
		} else {
			next, err = next.ClosestPrecedingFingerRPC(ID)
		}
		if err != nil {
			return nil, err
		}
		succ, err = next.GetSuccessorRPC()
		if err != nil {
			return nil, err
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
		if inInterval(fingerNode.ID, incID(n.ID), ID) {
			return fingerNode
		}
	}
	return n
}

// Leave the DHT ring and stop the node
func (n *Node) Leave() error {
	return n.shutDown()
	// TODO: any other tasks?
	// inform other nodes or copy keys to them before leaving?
}

func (n *Node) shutDown() error {
	close(n.shutdownCh)
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

// PrintFT prints the finger table of node n
func (n *Node) PrintFT() {
	for i, ent := range n.ft {
		fmt.Printf("%d %s %s\n", i, ent.start, ent.node.ID)
	}
}

func genID(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	hash := hex.EncodeToString(h.Sum(nil))
	return hash
}

// func genID(s string) string {
// 	m := map[string]string{"localhost:5554": "01", "localhost:5555": "05",
// 		"localhost:5556": "0a", "localhost:5557": "0f"}
// 	return m[s]
// }

// const IDChars int = 2
// const IDBits = 5
