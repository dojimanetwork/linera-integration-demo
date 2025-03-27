package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

func generateJWTSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func main() {
	// Generate a 32-byte (256-bit) secret key
	secret, err := generateJWTSecret(32)
	if err != nil {
		fmt.Printf("Error generating secret: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated JWT Secret: %s\n", secret)
	fmt.Println("\nAdd this to your environment variables:")
	fmt.Printf("export JWT_SECRET='%s'\n", secret)
}