package dht

import (
	"log"
	"sync"
)

type transport struct {
	remotes map[string]*Client
	node    *Node
	mux     *sync.RWMutex
}

func newTransport(node *Node, remotes map[string]*Client, mux *sync.RWMutex) *transport {
	if remotes == nil {
		remotes = make(map[string]*Client)
	}
	if mux == nil {
		mux = &sync.RWMutex{}
	}
	t := &transport{node: node, remotes: remotes, mux: mux}
	return t
}

func (t *transport) getRemote() *Client {
	cli := t.getRemoteIfExists()
	if cli != nil {
		return cli
	}
	return t.setRemote()
}

func (t *transport) getRemoteIfExists() *Client {
	t.mux.RLock()
	cli, ok := t.remotes[t.node.Addr]
	t.mux.RUnlock()
	if ok {
		return cli
	} else {
		return nil
	}
}

func (t *transport) setRemote() *Client {
	cli := t.setRemoteWithCredentials(t.node.Addr, false, "", "")
	t.mux.Lock()
	t.remotes[t.node.Addr] = cli
	t.mux.Unlock()
	return cli
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
	res, err := cli.GetSuccessor()
	return NewRemoteNode(res.GetAddr(), res.GetId(), t), err
}

func (t *transport) getPredecessor() (*Node, error) {
	cli := t.getRemote()
	res, err := cli.GetPredecessor()
	return NewRemoteNode(res.GetAddr(), res.GetId(), t), err
}

func (t *transport) setPredecessor(node *Node) error {
	cli := t.getRemote()
	return cli.SetPredecessor(node)
}

func (t *transport) findSuccessor(id string) (*Node, error) {
	cli := t.getRemote()
	res, err := cli.FindSuccessor(id)
	return NewRemoteNode(res.GetAddr(), res.GetId(), t), err
}

func (t *transport) closestPrecedingFinger(id string) (*Node, error) {
	cli := t.getRemote()
	res, err := cli.ClosestPrecedingFinger(id)
	return NewRemoteNode(res.GetAddr(), res.GetId(), t), err
}

func (t *transport) close() {
	cli := t.getRemoteIfExists()
	if cli != nil {
		cli.Close()
		t.mux.Lock()
		delete(t.remotes, t.node.Addr)
		t.mux.Unlock()
	}
}
