package bootconfig

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/systemboot/systemboot/pkg/crypto"
	"golang.org/x/crypto/ed25519"
)

// memoryZipReader is used to unpack a zip file from a byte sequence in memory.
type memoryZipReader struct {
	Content []byte
}

func (r *memoryZipReader) ReadAt(p []byte, offset int64) (n int, err error) {
	cLen := int64(len(r.Content))
	if offset > cLen {
		return 0, io.EOF
	}
	if cLen-offset >= int64(len(p)) {
		n = len(p)
		err = nil
	} else {
		err = io.EOF
		n = int(int64(cLen) - offset)
	}
	copy(p, r.Content[offset:int(offset)+n])
	return n, err
}

// FromZip tries to extract a boot configuration from a ZIP file after verifying
// its signature with the provided public key file. The signature is expected to
// be appended to the ZIP file and have fixed length `ed25519.SignatureSize` .
// The returned string argument is the temporary directory where the files were
// extracted, if successful.
// No decoder (e.g. JSON, ZIP) or other function parsing the input file is called
// before verifying the signature.
func FromZip(filename string, pubkeyfile *string) (*Manifest, string, error) {
	// load the whole zip file in memory - we need it anyway for the signature
	// matching.
	// TODO refuse to read if too big?
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, "", err
	}
	zipbytes := data
	// Load the public key
	if pubkeyfile != nil {
		zipbytes = data[:len(data)-ed25519.SignatureSize]
		pubkey, err := crypto.LoadPublicKeyFromFile(*pubkeyfile)
		if err != nil {
			return nil, "", err
		}

		// Load the signature.
		// The signature is appended to the zip file and has length
		// `ed25519.SignatureSize`. We read these bytes from the end of the file and
		// treat them as the attached signature.
		signature := data[len(data)-ed25519.SignatureSize : len(data)]
		if len(signature) != ed25519.SignatureSize {
			return nil, "", fmt.Errorf("Short read when reading signature: want %d bytes, got %d", ed25519.SignatureSize, len(signature))
		}

		// Verify the signature against the public key and the zip file bytes
		if ok := ed25519.Verify(pubkey, zipbytes, signature); !ok {
			return nil, "", fmt.Errorf("Invalid ed25519 signature for file %s", filename)
		}
	} else {
		log.Printf("No public key specified, the ZIP file will be unpacked without verification")
	}

	// At this point the signature is valid. Unzip the file and decode the boot
	// configuration.
	r, err := zip.NewReader(&memoryZipReader{Content: zipbytes}, int64(len(zipbytes)))
	if err != nil {
		return nil, "", err
	}
	tempDir, err := ioutil.TempDir(os.TempDir(), "bootconfig")
	if err != nil {
		return nil, "", err
	}
	log.Printf("Created temporary directory %s", tempDir)
	var manifest *Manifest
	for _, f := range r.File {
		destination := path.Join(tempDir, f.Name)
		// NOTE I don't know if it's possible to have a zero-length file name in
		// a zip file. However I am not checking for this case as I prefer this
		// to blow up with a panic and highlight the unexpected malformed file.
		if f.Name[len(f.Name)-1] == '/' {
			// it's a directory, create it
			if err := os.MkdirAll(destination, os.ModeDir|os.FileMode(0700)); err != nil {
				return nil, "", err
			}
			log.Printf("Extracted directory %s (flags: %d)", f.Name, f.Flags)
		} else {
			fd, err := f.Open()
			if err != nil {
				return nil, "", err
			}
			buf, err := ioutil.ReadAll(fd)
			if err != nil {
				return nil, "", err
			}
			if f.Name == "manifest.json" {
				// parse the Manifest containing the boot configurations
				manifest, err = NewManifest(buf)
				if err != nil {
					return nil, "", err
				}
			}
			// FIXME how to get permissions stored in the zip file?
			perm := os.FileMode(0644)
			if err := ioutil.WriteFile(destination, buf, perm); err != nil {
				return nil, "", err
			}
			log.Printf("Extracted file %s (flags: %d)", f.Name, f.Flags)
		}
	}
	if manifest == nil {
		return nil, "", errors.New("No manifest found")
	}
	return manifest, tempDir, nil
}
