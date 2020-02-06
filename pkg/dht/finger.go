package dht

import (
	"math"
	"math/big"
)

type fingerEntry struct {
	node  *Node
	start string
}

func initFT(node *Node) []fingerEntry {
	ft := make([]fingerEntry, IDBits)
	for i := range ft {
		ft[i] = fingerEntry{start: getFEStart(node.ID, i)}
	}
	return ft
}

func getFEStart(n string, idx int) string {
	startInt, _ := new(big.Int).SetString(n, 16)            // n
	twoToI := big.NewInt(int64(math.Exp2(float64(idx))))    // 2^i
	startInt.Add(startInt, twoToI)                          // (n + 2^i)
	twoToM := big.NewInt(int64(math.Exp2(float64(IDBits)))) // 2^m
	startInt.Mod(startInt, twoToM)                          // (n + 2^i) mod 2^m
	return appendZeros(startInt.Text(16), IDChars)
}

// fillFT fills all entries of FT except for first one
// must fill successor value first
func fillFT(n *Node, helperNode *Node) error {
	// first entry must have been set before to successor
	for i := 1; i < len(n.ft); i++ {
		if inInterval(n.ft[i].start, n.ID, n.ft[i-1].node.ID) {
			n.ft[i].node = n.ft[i-1].node
		} else {
			succ, err := helperNode.FindSuccessorRPC(n.ft[i].start)
			if err != nil {
				return err
			}
			n.ft[i].node = succ
		}
	}
	return nil
}

func fillFTFirstNode(node *Node) {
	for i := range node.ft {
		node.ft[i].node = node
	}
}

// inInterval checks if key is in interval [start, end)
// where key, start, and end are in hexadecimal encoding
func inInterval(key string, start string, end string) bool {
	if start == end {
		return false
	}
	if start < end {
		if (key >= start) && (key < end) {
			return true
		}
	} else {
		// 0 is somewhere between start and end
		if (key >= start) || (key < end) {
			return true
		}
	}
	return false
}

func incID(id string) string {
	idInt, _ := new(big.Int).SetString(id, 16)
	idInt.Add(idInt, big.NewInt(1))
	return appendZeros(idInt.Text(16), IDChars)
}

func appendZeros(s string, length int) string {
	for len(s) < length {
		s = "0" + s
	}
	return s
}
