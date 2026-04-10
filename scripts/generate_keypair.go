// +build ignore

// Standalone script to generate an Ed25519 keypair for custom client signing.
// Usage: go run scripts/generate_keypair.go
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate keypair: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("=== Ed25519 Keypair for Custom Client ===")
	fmt.Println()
	fmt.Println("Public Key (put in rustdesk/src/common.rs line 2187):")
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
	fmt.Println()
	fmt.Println("Private Key (put in conf/config.yaml custom-client.signing-key):")
	fmt.Println(base64.StdEncoding.EncodeToString(priv))
}
