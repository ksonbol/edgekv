package dht

import (
	"crypto/sha1"
	"fmt"
)

const hashBits = 128

// type DHT interface {
// 	Successor(k string) *Router
// 	Join(node Router) error
// 	Leave() error
// }

// Node represents a node in the DHT overlay
type Node struct {
	ID   string
	Hash string
	// Transport *dhtServer
	ft   []fingerEntry
	pred *Node
}

// NewNode creates a new DHT node, initializing hash and transport
func NewNode(ID string) *Node {
	n := &Node{ID: ID}
	n.Hash = genHash(ID)
	n.ft = initFT(n)
	// TODO: initialize transport
	return n
}

// Join node to a DHT with the help of helperNode
func (n *Node) Join(helperNode *Node) error {
	if helperNode == nil {
		// first node in dht
		fillFTFirstNode(n)
		n.pred = n
	} else {
		// TODO: implement find successor RPC
		succ, err := helperNode.findSuccessor(n.ft[0].start)
		if err != nil {
			return err
		}
		n.ft[0].node = succ
		n.pred = succ.Predecessor()
		succ.pred = n
		err = fillFT(n, helperNode)
		if err != nil {
			return err
		}
	}
	return nil
}

// Successor returns the successor of node n
func (n *Node) Successor() *Node {
	return n.ft[0].node
}

// Predecessor returns the predecessor of node n
func (n *Node) Predecessor() *Node {
	return n.pred
}

// FindSuccessor uses DHT RPCs to find the successor of a specific key (hash)
func (n *Node) findSuccessor(hash string) (*Node, error) {
	// TODO: implement this
	return n, nil
}

// Leave the DHT ring and stop the node
func (n *Node) Leave() error {
	return nil
}

func genHash(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}
