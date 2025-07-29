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
	fmt.Println("RSA Key Pair generated successfully!")
	publicKey := &privateKey.PublicKey
	pkcs8PrivateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Failed to marshal private key to PKCS#8: %v", err)
	}
	pkcs8Pem := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8PrivateKeyBytes,
	})
	spkiPublicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Fatalf("Failed to marshal public key to SPKI: %v", err)
	}
	spkiPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: spkiPublicKeyBytes,
	})
	err = os.WriteFile(savePath+"/public.pem", pkcs8Pem, 0644)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(savePath+"/private.pem", spkiPem, 0644)
	if err != nil {
		panic(err)
	}
	return spkiPem, pkcs8Pem
}
