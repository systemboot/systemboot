package bootconfig

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	bcOutputFile = "tests/bc.file"
	manifestFile = "tests/manifest.json"
	kernelsDir   = "tests/kernels"
	initrdsDir   = "tests/initrds"
	dtsDir       = "tests/dts"
)

var (
	// password is a PEM encrypted passphrase
	password       = []byte{'k', 'e', 'i', 'n', 's'}
	publicKeyFile  = "tests/keys/public_key.pem"
	privateKeyFile = "tests/keys/private_key.pem"
)

func getFilePathsByDir(dirName string) ([]string, error) {
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		return nil, err
	}

	var listOfFilePaths []string
	for _, file := range files {
		if !file.IsDir() {
			listOfFilePaths = append(listOfFilePaths, path.Join(dirName, file.Name()))
		}
	}

	return listOfFilePaths, nil
}

func TestPackBCSigned(t *testing.T) {
	kernelFiles, err := getFilePathsByDir(kernelsDir)
	require.NoError(t, err)

	initrdFiles, err := getFilePathsByDir(initrdsDir)
	require.NoError(t, err)

	dtFiles, err := getFilePathsByDir(dtsDir)
	require.NoError(t, err)

	err = Pack(bcOutputFile, manifestFile, kernelFiles, initrdFiles, dtFiles, &privateKeyFile, password)
	require.NoError(t, err)
}

func TestUnpackBCSigned(t *testing.T) {
	mc, dir, err := Unpack(bcOutputFile, &publicKeyFile)
	require.NoError(t, err)
	require.NotNil(t, dir)
	require.NotNil(t, mc)
}

func TestPackBCUnsigned(t *testing.T) {
	kernelFiles, err := getFilePathsByDir(kernelsDir)
	require.NoError(t, err)

	initrdFiles, err := getFilePathsByDir(initrdsDir)
	require.NoError(t, err)

	dtFiles, err := getFilePathsByDir(dtsDir)
	require.NoError(t, err)

	err = Pack(bcOutputFile, manifestFile, kernelFiles, initrdFiles, dtFiles, nil, nil)
	require.NoError(t, err)
}

func TestUnpackBCUnsigned(t *testing.T) {
	mc, dir, err := Unpack(bcOutputFile, nil)
	require.NoError(t, err)
	require.NotNil(t, dir)
	require.NotNil(t, mc)
}

func TestGetBootConfigByName(t *testing.T) {
	bc, dir, err := GetBootConfig(bcOutputFile, nil, "test2")
	require.NoError(t, err)
	require.NotNil(t, dir)
	require.NotNil(t, bc)
	require.Equal(t, bc.Name, "test2")
}
