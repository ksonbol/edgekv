package dht

import (
	"crypto/sha1"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/ksonbol/edgekv/utils"
)

// IDBits is the number of bits contained in ID hashes
const IDBits = 128

// Node represents a node in the DHT overlay
type Node struct {
	Addr       string
	ID         string
	ft         []fingerEntry
	transport  *transport
	pred       *Node
	server     *Server
	shutdownCh chan struct{}
}

// newBasicNode creates a node with the essential parameters
func newBasicNode(addr string, id string) *Node {
	return &Node{Addr: addr, ID: id, shutdownCh: make(chan struct{})}
}

// NewLocalNode creates a new DHT node, initializing ID and transport
func NewLocalNode(addr string) *Node {
	n := newBasicNode(addr, genID(addr))
	n.ft = initFT(n)
	hostname, port := utils.SplitAddress(addr)
	n.server = NewServer(hostname, port, n)
	n.server.RunInsecure()
	n.transport = newTransport(n, nil, nil)
	return n
}

// NewRemoteNode returns a Node instance without the FT
func NewRemoteNode(addr string, id string, localTransport *transport) *Node {
	if id == "" {
		id = genID(addr)
	}
	n := newBasicNode(addr, id)
	n.transport = newTransport(n, localTransport.remotes, localTransport.mux)
	return n
}

// Join node to a DHT with the help of helperNode
func (n *Node) Join(helperNode *Node) error {
	rand.Seed(time.Now().UnixNano())
	succ, err := helperNode.FindSuccessorRPC(n.ft[0].start)
	if err != nil {
		return err
	}
	n.SetSuccessor(succ)

	go n.stabilize()
	go n.fixFingers()
	return nil
}

// GetSuccessorRPC returns the successor of n using an RPC call
func (n *Node) GetSuccessorRPC() (*Node, error) {
	return n.transport.getSuccessor()
}

// GetPredecessorRPC returns the predecessor of n using an RPC call
func (n *Node) GetPredecessorRPC() (*Node, error) {
	return n.transport.getPredecessor()
}

// FindSuccessorRPC finds the successor of a key an RPC call to n
// this may initiate more RPC calls
func (n *Node) FindSuccessorRPC(id string) (*Node, error) {
	return n.transport.findSuccessor(id)
}

// ClosestPrecedingFingerRPC returns the closest node that precedes id
func (n *Node) ClosestPrecedingFingerRPC(id string) (*Node, error) {
	return n.transport.closestPrecedingFinger(id)
}

// NotifyRPC notify node that n should be their predecessor
func (n *Node) NotifyRPC(node *Node) error {
	return n.transport.notify(node)
}

// Successor returns the successor of node n
func (n *Node) Successor() *Node {
	return n.ft[0].node
}

// Predecessor returns the predecessor of node n
func (n *Node) Predecessor() *Node {
	return n.pred
}

// SetPredecessor sets the predecessor of node n
func (n *Node) SetPredecessor(pred *Node) {
	n.pred = pred
}

// SetSuccessor sets the successsor of node n in the ft
func (n *Node) SetSuccessor(succ *Node) {
	n.ft[0].node = succ
}

// stabilize periodically checks the successor links
func (n *Node) stabilize() {
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			x, err := n.Successor().GetPredecessorRPC()
			if err != nil {
				log.Fatalf("Failed to run the stabilizer algorithm %v", err)
			}
			// if x in (n, successor)
			if inIntervalHex(x.ID, incID(n.ID), n.Successor().ID) {
				n.SetSuccessor(x)
			}
			n.Successor().NotifyRPC(n)
		case <-n.shutdownCh:
			ticker.Stop()
			return
		}
	}
}

// fixFingers periodically updates a finger chosen at random
func (n *Node) fixFingers() {
	// todo run periodically
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
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
	succ, err := pred.GetSuccessorRPC()
	if err != nil {
		return nil, err
	}
	return succ, nil
}

// findPredecessor uses DHT RPCs to find the predecessor of a specific key (ID)
func (n *Node) findPredecessor(ID string) (*Node, error) {
	next := n
	err := *new(error)
	succ := next.Successor() // current node, no need for RPC yet
	// while id not in (next.ID, next.Successor.ID]
	for !inIntervalHex(ID, incID(next.ID), incID(succ.ID)) {
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
		if inIntervalHex(fingerNode.ID, incID(n.ID), ID) {
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
	for _, conn := range n.transport.remotes {
		conn.Close()
	}
	n.transport.shutdown()
	n.ft = nil
	n.pred = nil
	return nil
}

func genID(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
