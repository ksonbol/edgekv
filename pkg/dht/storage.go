package dht

// Storage represents the dht storage interface
type Storage interface {
	GetKV(key string) (string, error)
	PutKV(key, value string) error
	DelKV(key string) error
	RangeGetKV(start, end string) (map[string]string, error)
	RangeDelKV(start, end string) error
	Close() error
}
