package security

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	publicKeyDER  string = "tests/public_key.der"
	publicKeyPEM  string = "tests/public_key.pem"
	testData      string = "tests/data"
	signatureGood string = "tests/verify_rsa_pkcs15_sha256.signature"
	signatureBad  string = "tests/verify_rsa_pkcs15_sha256.signature2"
)

func TestLoadDERPublicKey(t *testing.T) {
	err := VerifyRsaSha256Pkcs1v15Signature(publicKeyDER, testData, signatureGood)
	require.Error(t, err)
}

func TestLoadPEMPublicKey(t *testing.T) {
	err := VerifyRsaSha256Pkcs1v15Signature(publicKeyPEM, testData, signatureGood)
	require.NoError(t, err)
}

func TestGoodSignature(t *testing.T) {
	err := VerifyRsaSha256Pkcs1v15Signature(publicKeyPEM, testData, signatureGood)
	require.NoError(t, err)
}

func TestBadSignature(t *testing.T) {
	err := VerifyRsaSha256Pkcs1v15Signature(publicKeyPEM, testData, signatureBad)
	require.Error(t, err)
}
