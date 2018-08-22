package bootconfig

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/mholt/archiver"
	"github.com/systemboot/systemboot/pkg/crypto"
	"golang.org/x/crypto/ed25519"
)

var (
	// DefaultTmpDir is the Linux default tmp directory
	DefaultTmpDir = "/tmp"
	// DefaultManifestJSONFilename in the boot configuration
	DefaultManifestJSONFilename = "manifest.json"
	// DefaultDeviceTreePath in the boot configuration
	DefaultDeviceTreePath = "device-tree"
	// DefaultKernelPath in the boot configuration
	DefaultKernelPath = "kernel"
	// DefaultInitrdPath in the boot configuration
	DefaultInitrdPath = "initrd"
)

// GetBootConfig returns a boot configuration with unpack directory for ĺater
// processing
func GetBootConfig(filename string, publicKeyPath *string, bootName string) (*BootConfig, *string, error) {
	mc, unpackDir, err := Unpack(filename, publicKeyPath)
	if err != nil {
		return nil, nil, err
	}

	for _, config := range mc.Configs {
		if config.Name == bootName {
			if len(config.DeviceTree) > 0 {
				config.DeviceTree = path.Join(*unpackDir, DefaultDeviceTreePath, config.DeviceTree)
			}
			if len(config.Kernel) > 0 {
				config.Kernel = path.Join(*unpackDir, DefaultKernelPath, config.Kernel)
			}
			if len(config.Initrd) > 0 {
				config.Initrd = path.Join(*unpackDir, DefaultInitrdPath, config.Initrd)
			}
			return &config, unpackDir, nil
		}
	}

	return nil, nil, errors.New("No matching configuration found")
}

// Unpack boot configuration
func Unpack(filename string, publicKeyPath *string) (*ManifestConfig, *string, error) {
	if !archiver.Zip.Match(filename) {
		return nil, nil, errors.New("File is not in Zip format")
	}

	unpackDir, err := ioutil.TempDir(DefaultTmpDir, "")
	if err != nil {
		return nil, nil, err
	}

	if err = archiver.Zip.Open(filename, unpackDir); err != nil {
		return nil, nil, err
	}

	filepath := path.Join(unpackDir, DefaultManifestJSONFilename)
	manifest, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, nil, err
	}

	if publicKeyPath != nil {
		signature, err := ioutil.ReadFile(filename)
		if err != nil {
			return nil, nil, err
		}

		publicKey, err := crypto.LoadPublicKeyFromFile(*publicKeyPath)
		if err != nil {
			return nil, nil, err
		}

		offset := len(signature) - ed25519.SignatureSize
		if err := ed25519.Verify(publicKey, signature[:offset], signature[offset:]); !err {
			return nil, nil, errors.New("signature verification of boot configuration file failed")
		}
	}

	mc := ManifestConfig{}
	if err := json.Unmarshal(manifest, &mc); err != nil {
		return nil, nil, err
	}

	return &mc, &unpackDir, nil
}

// Pack boot configuration
func Pack(outputFilePath string, manifestFilePath string, kernelFilePaths []string, initrdFilePaths []string, dtFilePaths []string, privateKeyPath *string, privateKeyPassword []byte) error {
	packDir, err := ioutil.TempDir(DefaultTmpDir, "")
	if err != nil {
		return err
	}

	manifest, err := ioutil.ReadFile(manifestFilePath)
	if err != nil {
		return err
	}

	mc := ManifestConfig{}
	if err = json.Unmarshal(manifest, &mc); err != nil {
		return err
	}

	manifestPath := path.Join(packDir, DefaultManifestJSONFilename)
	if err = ioutil.WriteFile(manifestPath, manifest, 777); err != nil {
		return err
	}

	kernelPath := path.Join(packDir, DefaultKernelPath)
	if err = os.MkdirAll(kernelPath, 0700); err != nil {
		return err
	}
	for _, kernel := range kernelFilePaths {
		kernelData, err := ioutil.ReadFile(kernel)
		if err != nil {
			return err
		}

		newKernelFilePath := path.Join(kernelPath, filepath.Base(kernel))
		if err = ioutil.WriteFile(newKernelFilePath, kernelData, 777); err != nil {
			return err
		}
	}

	initrdPath := path.Join(packDir, DefaultInitrdPath)
	if err := os.MkdirAll(initrdPath, 0700); err != nil {
		return err
	}
	for _, initrd := range initrdFilePaths {
		initrdData, err := ioutil.ReadFile(initrd)
		if err != nil {
			return err
		}

		newInitrdFilePath := path.Join(initrdPath, filepath.Base(initrd))
		if err = ioutil.WriteFile(newInitrdFilePath, initrdData, 777); err != nil {
			return err
		}
	}

	dtPath := path.Join(packDir, DefaultDeviceTreePath)
	if err := os.MkdirAll(dtPath, 0700); err != nil {
		return err
	}
	for _, dt := range dtFilePaths {
		dtData, err := ioutil.ReadFile(dt)
		if err != nil {
			return err
		}

		newDtFilePath := path.Join(dtPath, filepath.Base(dt))
		if err = ioutil.WriteFile(newDtFilePath, dtData, 777); err != nil {
			return err
		}
	}

	var files []string
	files = append(files, manifestPath, kernelPath, initrdPath, dtPath)
	if err := archiver.Zip.Make(outputFilePath, files); err != nil {
		return err
	}

	if privateKeyPath != nil {
		tarFile, err := ioutil.ReadFile(outputFilePath)
		if err != nil {
			return err
		}

		privateKey, err := crypto.LoadPrivateKeyFromFile(*privateKeyPath, privateKeyPassword)
		if err != nil {
			return err
		}

		signature := ed25519.Sign(privateKey, tarFile)
		if len(signature) <= 0 {
			return errors.New("signing boot configuration failed")
		}

		tarFile = append(tarFile, signature...)
		if err = ioutil.WriteFile(outputFilePath, tarFile, 777); err != nil {
			return err
		}
	}

	return os.RemoveAll(packDir)
}
