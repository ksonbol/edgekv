package dht

import (
	"log"
)

type transport struct {
	remote *Client
	node   *Node
}

func newTransport(node *Node) *transport {
	t := &transport{node: node}
	return t
}

func (t *transport) getRemote() *Client {
	if t.remote != nil {
		return t.remote
	}
	return t.setRemote()
}

func (t *transport) setRemote() *Client {
	return t.setRemoteWithCredentials(t.node.Addr, false, "", "")
}

func (t *transport) setRemoteWithCredentials(serverAddr string, tls bool, caFile string,
	serverHostOverride string) *Client {
	cli, err := NewClient(serverAddr, tls, caFile, serverHostOverride)
	if err != nil {
		log.Fatalf("Failed to create connection with remote server %v", err)
	}
	return cli
}

func (t *transport) getSuccessor() (*Node, error) {
	cli := t.getRemote()
	return cli.GetSuccessor()
}

func (t *transport) getPredecessor() (*Node, error) {
	cli := t.getRemote()
	return cli.GetPredecessor()
}

func (t *transport) setPredecessor(node *Node) error {
	cli := t.getRemote()
	return cli.SetPredecessor(node)
}

func (t *transport) findSuccessor(id string) (*Node, error) {
	cli := t.getRemote()
	return cli.FindSuccessor(id)
}

func (t *transport) closestPrecedingFinger(id string) (*Node, error) {
	cli := t.getRemote()
	return cli.ClosestPrecedingFinger(id)
}
