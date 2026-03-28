package auth

// KeychainGet retrieves a credential from the platform keychain.
// This is a stub implementation; production builds use platform-specific build tags.
func KeychainGet(service, account string) (string, error) {
	return "", nil
}

// KeychainSet stores a credential in the platform keychain.
// This is a stub implementation; production builds use platform-specific build tags.
func KeychainSet(service, account, value string) error {
	return nil
}
