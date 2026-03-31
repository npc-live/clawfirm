package keychain

// Keychain abstracts secure key storage (macOS Keychain, Linux file fallback).
type Keychain interface {
	GetKey(service string) ([]byte, error)
	SetKey(service string, key []byte) error
}
