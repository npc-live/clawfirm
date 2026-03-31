package vault

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// ReadSecret reads a secret from the terminal without echoing it.
// prompt is printed before reading.
func ReadSecret(prompt string) ([]byte, error) {
	fmt.Fprint(os.Stderr, prompt)
	secret, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // newline after hidden input
	if err != nil {
		return nil, fmt.Errorf("read secret: %w", err)
	}
	return secret, nil
}
