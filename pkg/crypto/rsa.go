package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
)

var (
	// RSAKeyLength is the default RSA key length
	RSAKeyLength = 4096
	// PubKeyIdentifier is the PEM public key identifier
	PubKeyIdentifier = "PUBLIC KEY"
	// PrivKeyIdentifier is the PEM private key identifier
	PrivKeyIdentifier = "PRIVATE KEY"
	// PEMCipher is the PEM encryption algorithm
	PEMCipher = x509.PEMCipherAES256
	// PubKeyFilePermissions are the public key file perms
	PubKeyFilePermissions os.FileMode = 0644
	// PrivKeyFilePermissions are the private key file perms
	PrivKeyFilePermissions os.FileMode = 0600
)

// LoadPublicKeyFromFile loads DER formatted RSA public key from file.
func LoadPublicKeyFromFile(publicKeyPath string) (*rsa.PublicKey, error) {
	x509PEM, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		return nil, err
	}

	// Parse x509 PEM file
	block, _ := pem.Decode(x509PEM)
	if block == nil || block.Type != PubKeyIdentifier {
		return nil, errors.New("Can't decode PEM file")
	}

	// parse public Key
	public, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return public, nil
}

// LoadPrivateKeyFromFile loads PKCS1 PEM formatted RSA private key from file.
func LoadPrivateKeyFromFile(privateKeyPath string, password []byte) (*rsa.PrivateKey, error) {
	x509PEM, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}

	// Parse x509 PEM file
	block, _ := pem.Decode(x509PEM)
	if block == nil || block.Type != PrivKeyIdentifier {
		return nil, errors.New("Can't decode PEM file")
	}

	// Check for encrypted PEM format
	var private *rsa.PrivateKey
	if x509.IsEncryptedPEMBlock(block) {
		decryptedKey, err := x509.DecryptPEMBlock(block, password)
		if err != nil {
			return nil, err
		}

		private, err = x509.ParsePKCS1PrivateKey(decryptedKey)
		if err != nil {
			return nil, err
		}
	} else {
		private, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	}

	return private, nil
}

// VerifyRsaSha256Pkcs1v15Signature verifies a PKCSv1.5 signature made by
// a SHA-256 checksum. Public key must be a RSA key in PEM format.
func VerifyRsaSha256Pkcs1v15Signature(publicKey *rsa.PublicKey, data []byte, signature []byte) error {
	if publicKey == nil {
		return errors.New("Couldn't import public key")
	}

	hash := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature)
}

// SignRsaSha256Pkcs1v15Signature signs data with a RSA private key, SHA-256
// for verification and a PKCSv1.5 padding.
func SignRsaSha256Pkcs1v15Signature(privateKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("Couldn't import private key")
	}

	rng := rand.Reader
	hash := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rng, privateKey, crypto.SHA256, hash[:])
}

// GenerateRSAKeys generates a PKCS1 RSA keypair
func GenerateRSAKeys(password []byte, privateKeyFilePath string, publicKeyFilePath string) error {
	key, err := rsa.GenerateKey(rand.Reader, RSAKeyLength)
	if err != nil {
		return err
	}

	var privKey = &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	var pubKey = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey),
	}

	var privateKey []byte
	if password != nil {
		encrypted, err := x509.EncryptPEMBlock(rand.Reader, privKey.Type, privKey.Bytes, password, PEMCipher)
		if err != nil {
			return err
		}
		privateKey = pem.EncodeToMemory(encrypted)
	} else {
		privateKey = pem.EncodeToMemory(privKey)
	}

	if err := ioutil.WriteFile(privateKeyFilePath, privateKey, PrivKeyFilePermissions); err != nil {
		return err
	}

	return ioutil.WriteFile(publicKeyFilePath, pem.EncodeToMemory(pubKey), PubKeyFilePermissions)
}
