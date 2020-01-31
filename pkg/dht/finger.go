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
	return startInt.Text(16)
}

// fillFT fills all entries of FT except for first one
// must fill successor value first
func fillFT(n *Node, helperNode *Node) error {
	// first entry must have been set before to successor
	for i := 1; i < len(n.ft); i++ {
		if inIntervalHex(n.ft[i].start, n.ID, n.ft[i-1].node.ID) {
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
func inInterval(key *big.Int, start *big.Int, end *big.Int) bool {
	keyText := key.Text(10)
	stText := start.Text(10)
	endText := end.Text(10)
	if stText < endText {
		if (keyText >= stText) && (keyText < endText) {
			return true
		}
	} else {
		// 0 is somewhere between start and end
		if (keyText >= stText) || (keyText < endText) {
			return true
		}
	}
	return false
}

// inIntervalHex checks if key is in interval [start, end)
// where key, start, and end are in hexadecimal notation
func inIntervalHex(key string, start string, end string) bool {
	keyInt, _ := new(big.Int).SetString(key, 16)
	startInt, _ := new(big.Int).SetString(start, 16)
	endInt, _ := new(big.Int).SetString(end, 16)
	return inInterval(keyInt, startInt, endInt)
}

func incID(id string) string {
	idInt, _ := new(big.Int).SetString(id, 16)
	idInt.Add(idInt, big.NewInt(1))
	return idInt.Text(16)
}
