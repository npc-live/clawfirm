package store

import "errors"

// ErrNotFound is returned when a key does not exist.
var ErrNotFound = errors.New("key not found")

// Store is a simple key-value store with byte slice values.
type Store interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	List() ([]string, error)
	Close() error
}
