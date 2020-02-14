package dht

import (
	"testing"
	"time"
)

func shortID(s string) string {
	m := map[string]string{"localhost:5554": "001", "localhost:5555": "005",
		"localhost:5556": "00a", "localhost:5557": "00f"}
	return m[s]
	// return genID(s)[0:2]
}

func TestFingerTables(t *testing.T) {
	conf := &Config{IDBits: 5, IDChars: 3, IDFunc: shortID}
	nodeAddr := map[int]string{1: "localhost:5554", 2: "localhost:5555", 3: "localhost:5556", 4: "localhost:5557"}
	node := NewLocalNode(nodeAddr[1], nil, conf)
	node2 := NewLocalNode(nodeAddr[2], nil, conf)
	node3 := NewLocalNode(nodeAddr[3], nil, conf)
	node4 := NewLocalNode(nodeAddr[4], nil, conf)
	m := map[int]*Node{1: node, 2: node2, 3: node3, 4: node4}
	helperNode2 := NewRemoteNode(node.Addr, node.ID, node2.Transport, conf)
	helperNode3 := NewRemoteNode(node.Addr, node.ID, node3.Transport, conf)
	helperNode4 := NewRemoteNode(node.Addr, node.ID, node4.Transport, conf)
	time.Sleep(2 * time.Second)
	node.Join(nil)
	node2.Join(helperNode2)
	node3.Join(helperNode3)
	node4.Join(helperNode4)
	time.Sleep(5 * time.Second)
	ft := make(map[int][]string)
	ft[1] = []string{"005", "005", "005", "00a", "001"}
	ft[2] = []string{"00a", "00a", "00a", "00f", "001"}
	ft[3] = []string{"00f", "00f", "00f", "001", "001"}
	ft[4] = []string{"001", "001", "001", "001", "001"}
	for i, f := range ft {
		n := m[i]
		// fmt.Printf("Node %d\n with Addr %s, ID: %s\n", i, n.Addr, n.ID)
		n.PrintFT()
		// fmt.Printf("Successor: %s, Predecessor: %s\n", n.Successor().ID, n.Predecessor().ID)
		for j, fte := range f {
			res := n.GetFTID(j)
			want := fte
			if res != want {
				t.Errorf("FT%d[%d] = %s, want %s", i, j, res, want)
			}
		}
	}
}
