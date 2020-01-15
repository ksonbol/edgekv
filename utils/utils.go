package utils

import "fmt"

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
