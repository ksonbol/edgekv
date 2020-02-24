package gateway

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

type HashMapStorage struct {
	m    map[string]string
	mMut sync.RWMutex
}

func NewHashMapStorage() *HashMapStorage {
	return &HashMapStorage{m: make(map[string]string)}
}

func (s *HashMapStorage) GetKV(key string) (string, error) {
	// hash := GenID(key)
	s.mMut.RLock()
	val, prs := s.m[key]
	// fmt.Printf("Get V(%s)=%s\n", key, val)
	s.mMut.RUnlock()
	if prs {
		return val, nil
	}
	return "", fmt.Errorf("Key %s does not exist\n", key)
}

func (s *HashMapStorage) PutKV(key, value string) error {
	// hash := GenID(key)
	s.mMut.Lock()
	s.m[key] = value
	// fmt.Printf("Put V(%s)=%s\n", key, value)
	s.mMut.Unlock()
	return nil
}

func (s *HashMapStorage) DelKV(key string) error {
	// hash := GenID(key)
	s.mMut.Lock()
	delete(s.m, key)
	s.mMut.Unlock()
	return nil
}

// for internal dht usage, not for clients
func (s *HashMapStorage) RangeGetKV(start, end string) (map[string]string, error) {
	// start and end are already hashed
	res := make(map[string]string)
	max := MaxID()
	zero := ZeroID()
	s.mMut.RLock()
	for k := range s.m {
		if ((start <= end) && (k >= start) && (k < end)) ||
			((start > end) && (((k >= start) && (k <= max)) || ((k >= zero) && (k < end)))) {
			// fmt.Printf("Range: Get V(%s)=%s\n", k, s.m[k])
			res[k] = s.m[k]
		}
	}

	s.mMut.RUnlock()
	return res, nil
}

// not supported
func (s *HashMapStorage) RangeDelKV(start, end string) error {
	max := MaxID()
	zero := ZeroID()
	s.mMut.Lock()
	for k := range s.m {
		if ((start <= end) && (k >= start) && (k < end)) ||
			((start > end) && (((k >= start) && (k <= max)) || ((k >= zero) && (k < end)))) {
			// fmt.Printf("Range: Del V(%s)\n", k)
			delete(s.m, k)
		}
	}
	s.mMut.Unlock()
	return nil
}

// MultiPut is used only by the dht node, not the clients
// it assumes given keys are already hashed
func (s *HashMapStorage) MultiPutKV(kvs map[string]string) error {
	s.mMut.Lock()
	for k, v := range kvs {
		// fmt.Printf("MultiPut: Put V(%s)=%s\n", k, v)
		s.m[k] = v
	}
	s.mMut.Unlock()
	return nil
}

func (s *HashMapStorage) Close() error {
	s.mMut.Lock()
	s.m = nil
	s.mMut.Unlock()
	return nil
}

// ZeroID return the Zero ID in the ring
func ZeroID() string {
	return strings.Repeat("0", 40)
}

// MaxID return the maximum possible ID in the ring
func MaxID() string {
	return strings.Repeat("f", 40) // in hex
}

func GenID(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	hash := hex.EncodeToString(h.Sum(nil))
	return hash
}
