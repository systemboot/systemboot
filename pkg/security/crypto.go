package security

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
)

const (
	pubKeyIdentifier string = "PUBLIC KEY"
)

func loadPublicKeyFromFile(publicKeyPath string) (*rsa.PublicKey, error) {
	// Load pub key from x509 PEM file
	x509PEM, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}

	// Parse x509 PEM file
	block, _ := pem.Decode(x509PEM)
	if block == nil || block.Type != pubKeyIdentifier {
		return nil, err
	}

	// parse Public Key
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		return nil, err
	}
}

// VerifyRsaSha256Pkcs1v15Signature verifies a PKCSv1.5 signature made by
// a SHA-256 checksum. Public key must be a RSA key in PEM format.
func VerifyRsaSha256Pkcs1v15Signature(publicKeyPath string, dataFilePath string, signatureFilePath string) error {
	dataFile, err := ioutil.ReadFile(dataFilePath)
	if err != nil {
		return err
	}

	signatureFile, err := ioutil.ReadFile(signatureFilePath)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(dataFile)
	pubKey, err := loadPublicKeyFromFile(publicKeyPath)
	if pubKey == nil {
		return err
	}

	return rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], signatureFile)
}
