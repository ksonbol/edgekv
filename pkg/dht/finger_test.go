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
	for _, s := range tests {
		want = strings.Repeat("0", IDChars-len(s.want)) + s.want
		res = incID(s.in)
		if res != want {
			t.Errorf("incID(%s) = %s, wanted %s", s.in, res, want)
		}
	}
}
