package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// // DataType of the data stored in the kv-store
// type DataType bool

// // StatusCode for RPC responses
// type StatusCode int32

// Types of data in the system
const (
	LocalData        bool   = true
	GlobalData       bool   = false
	LocalDataStr     string = "local"
	GlobalDataStr    string = "global"
	KVAddedOrUpdated int32  = 0
	KVDeleted        int32  = 1
	KeyNotFound      int32  = 2
	UnknownError     int32  = 3
)

// KeyNotFoundError shows missing key in database
type KeyNotFoundError struct {
	Key string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("Key not found: %s", e.Key)
}

// GetHostname return the hostname/IP part of an address string
func GetHostname(addr string) string {
	return strings.Split(addr, ":")[0]
}

// GetPort returns the port number part of an address string
func GetPort(addr string) int {
	s := strings.Split(addr, ":")
	port, _ := strconv.Atoi(s[1])
	return port
}

// SplitAddress extracts and returns (hostname, port) from address string
func SplitAddress(addr string) (string, int) {
	s := strings.Split(addr, ":")
	hostname := s[0]
	port, _ := strconv.Atoi(s[1])
	return hostname, port
}
