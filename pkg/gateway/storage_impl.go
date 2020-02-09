package gateway

import (
	"github.com/ksonbol/edgekv/pkg/client"
	"github.com/ksonbol/edgekv/utils"
)

// EdgeKVStorage is a storage abstraction using edgeKV edge storage
// It implements the dht.Storage interface
type EdgeKVStorage struct {
	cl *client.EdgekvClient
}

// NewStorage creates a new instance of EdgeKVStorage with given client
func NewStorage(cl *client.EdgekvClient) *EdgeKVStorage {
	return &EdgeKVStorage{cl: cl}
}

// GetKV gets KV from the connected edge group
func (s *EdgeKVStorage) GetKV(key string) (string, error) {
	return s.cl.Get(key, utils.GlobalDataStr)
}

// PutKV puts the KV to the connected edge group
func (s *EdgeKVStorage) PutKV(key, value string) error {
	return s.cl.Put(key, utils.GlobalDataStr, value)
}

// DelKV removes the KV from the connected edge group
func (s *EdgeKVStorage) DelKV(key string) error {
	return s.cl.Del(key, utils.GlobalDataStr)
}

// Close closes the client connection to the connected edge group
func (s *EdgeKVStorage) Close() error {
	return s.cl.Close()
}
