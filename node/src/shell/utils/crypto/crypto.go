package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
)

func SecureUniqueString() string {
	return uuid.New().String() + "-" + uuid.New().String()
}

func SecureUniqueId(fed string) string {
	return uuid.New().String() + "@" + fed
}

func SecureKeyPairs(savePath string) ([]byte, []byte) {

	os.MkdirAll(savePath, os.ModePerm)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate RSA private key: %v", err)
	}
	publicKey := &privateKey.PublicKey
	fmt.Println("RSA Key Pair generated successfully.")
	fmt.Println("\n--- Step 2.1: Convert Private Key to PKCS#8 PEM String ---")
	pkcs8PrivateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Failed to marshal private key to PKCS#8 DER: %v", err)
	}
	pkcs8PrivateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8PrivateKeyDER,
	})
	fmt.Println("Generated PKCS#8 Private Key (PEM):")
	fmt.Println(string(pkcs8PrivateKeyPEM))
	fmt.Println("\n--- Step 2.2: Convert Public Key to SPKI PEM String ---")
	spkiPublicKeyDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Fatalf("Failed to marshal public key to SPKI DER: %v", err)
	}
	spkiPublicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY", // The standard PEM block type for SPKI public keys
		Bytes: spkiPublicKeyDER,
	})
	fmt.Println("Generated SPKI Public Key (PEM):")
	fmt.Println(string(spkiPublicKeyPEM))
	err = os.WriteFile(savePath+"/public.pem", spkiPublicKeyPEM, 0644)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(savePath+"/private.pem", pkcs8PrivateKeyPEM, 0644)
	if err != nil {
		panic(err)
	}
	return spkiPublicKeyPEM, pkcs8PrivateKeyPEM
}
