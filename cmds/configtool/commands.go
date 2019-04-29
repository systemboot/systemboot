package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"

	"github.com/systemboot/systemboot/pkg/bootconfig"
	"github.com/systemboot/systemboot/pkg/crypto"
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

// GenKeys generates ED25519 keypair and stores it on the harddrive
func GenKeys() error {
	return crypto.GeneratED25519Key([]byte(*genkeysPassphrase), *genkeysPrivateKeyFile, *genkeysPublicKeyFile)
}

// PackBootConfiguration packages a boot configuration containing different
// binaries and a manifest
func PackBootConfiguration() error {
	var err error
	var kernelFilePaths []string
	if *packKernelsDir != "" {
		kernelFilePaths, err = getFilePathsByDir(*packKernelsDir)
		if err != nil {
			log.Println("No kernels found")
		}
	}

	var initrdFilePaths []string
	if *packInitrdsDir != "" {
		initrdFilePaths, err = getFilePathsByDir(*packInitrdsDir)
		if err != nil {
			log.Println("No initrds found")
		}
	}

	var dtFilePaths []string
	if *packDTsDir != "" {
		dtFilePaths, err = getFilePathsByDir(*packDTsDir)
		if err != nil {
			log.Println("No device trees found")
		}
	}

	return bootconfig.ToZip(*packOutputFilename, *packManifest, kernelFilePaths, initrdFilePaths, dtFilePaths, packSignPrivateKeyFile, []byte(*packSignPassphrase))
}

// UnpackBootConfiguration unpacks a boot configuration file and returns the
// file path of a directory containing the data
func UnpackBootConfiguration() error {
	if *unpackDir != "" {
		// FIXME
		//bootconfig.DefaultTmpDir = *unpackDir
		fmt.Println(`flag "output-dir" currently not supported`)
	}

	if *unpackVerifyPublicKeyFile == "" {
		// FIXME
		// don't know how to handel it.
		// FromZip expects that no key is provided, only if pointer is nil
		unpackVerifyPublicKeyFile = nil
	}

	_, outputDir, err := bootconfig.FromZip(*unpackInputFilename, unpackVerifyPublicKeyFile)
	if err != nil {
		return err
	}

	fmt.Println("Boot configuration unpacked into: " + outputDir)

	return nil
}
