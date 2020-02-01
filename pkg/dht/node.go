package dht

import (
	"crypto/sha1"
	"fmt"
	"github.com/ksonbol/edgekv/utils"
)

// IDBits is the number of bits contained in ID hashes
const IDBits = 128

// type DHT interface {
// 	Successor(k string) *Router
// 	Join(node Router) error
// 	Leave() error
// }

// Node represents a node in the DHT overlay
type Node struct {
	Addr      string
	ID        string
	ft        []fingerEntry
	transport *transport
	pred      *Node
	server    *Server
}

// NewLocalNode creates a new DHT node, initializing ID and transport
func NewLocalNode(addr string) *Node {
	n := &Node{Addr: addr}
	n.ID = genID(addr)
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
	n := &Node{Addr: addr, ID: id}
	n.transport = newTransport(n, localTransport.remotes, localTransport.mux)
	return n
}

// Join node to a DHT with the help of helperNode
func (n *Node) Join(helperNode *Node) error {
	if helperNode == nil {
		// first node in dht
		fillFTFirstNode(n)
		n.SetPredecessor(n)
	} else {
		succ, err := helperNode.FindSuccessorRPC(n.ft[0].start)
		if err != nil {
			return err
		}
		n.ft[0].node = succ
		pred, err := succ.GetPredecessorRPC()
		if err != nil {
			return err
		}
		n.SetPredecessor(pred)
		// TODO: change this to an RPC call
		err = succ.SetPredecessorRPC(n)
		if err != nil {
			return err
		}
		err = fillFT(n, helperNode)
		if err != nil {
			return err
		}
		// TODO: update other nodes
		// TODO: copy keys from other nodes
	}
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

// SetPredecessorRPC sets the the predecessor of n using an RPC call
func (n *Node) SetPredecessorRPC(node *Node) error {
	return n.transport.setPredecessor(node)
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
	return nil
}

func genID(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
