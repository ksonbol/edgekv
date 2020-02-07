package dht

import (
	"math/big"
	"strings"
	"testing"
)

func TestInInterval(t *testing.T) {
	key := big.NewInt(20)
	start := big.NewInt(10)
	end := big.NewInt(30)
	keyHex := key.Text(16)
	startHex := start.Text(16)
	endHex := end.Text(16)
	var tests = []struct {
		a, b, c string
		want    bool
	}{
		{startHex, startHex, endHex, true},  // 10 in [10, 30)
		{keyHex, startHex, endHex, true},    // 20 in [10, 30)
		{endHex, startHex, endHex, false},   // 30 in [10, 30)
		{keyHex, keyHex, startHex, true},    // 20 in [20, 10)
		{endHex, keyHex, startHex, true},    // 30 in [20, 10)
		{startHex, keyHex, startHex, false}, // 10 in [20, 10)

	}
	var res bool
	for _, s := range tests {
		res = inInterval(s.a, s.b, s.c)
		if res != s.want {
			t.Errorf("inInterval(%s, %s, %s) = %v, wanted %v", s.a, s.b, s.c, res, s.want)
		}
	}
}

func TestIncID(t *testing.T) {
	var tests = []struct {
		in   string
		want string
	}{
		{"10", "11"},
		{"f", "10"},
		{"9f", "a0"},
		{"0", "1"},
		{"f1c", "f1d"},
	}
	var want string
	var res string
	idChars := 40
	for _, s := range tests {
		want = strings.Repeat("0", idChars-len(s.want)) + s.want
		res = incID(s.in, idChars)
		if res != want {
			t.Errorf("incID(%s) = %s, wanted %s", s.in, res, want)
		}
	}
}

func TestGetFEStart(t *testing.T) {
	var res string
	var tests = []struct {
		n                    string
		idx, idBits, idChars int
		want                 string
	}{
		{"001", 0, 5, 3, "002"}, // FT1[0] = succ(2)
		{"001", 1, 5, 3, "003"}, // FT1[1] = succ(3)
		{"001", 3, 5, 3, "009"},
		{"00a", 0, 5, 3, "00b"},
		{"00a", 3, 5, 3, "012"}, // FTa[3] = succ(18)
		{"00a", 4, 5, 3, "01a"},
	}
	for _, s := range tests {
		res = getFEStart(s.n, s.idx, s.idBits, s.idChars)
		if res != s.want {
			t.Errorf("getFEStart(%s, %d, %d, %d) = %s, wanted %s",
				s.n, s.idx, s.idBits, s.idChars, res, s.want)
		}
	}
}
