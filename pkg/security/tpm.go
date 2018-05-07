package security

import (
	"crypto/sha1"

	tspi "github.com/google/go-tpm/tpm"
)

const (
	tpmDefaultDevice string = "/dev/tpm0"
)

// ReadPcrTPM1 reads the PCR for the given
// index
func ReadPcrTPM1(pcr uint32) ([]byte, error) {
	rwc, err := tspi.OpenTPM(tpmDefaultDevice)
	if err != nil {
		return nil, err
	}

	defer rwc.Close()

	data, err := tspi.ReadPCR(rwc, pcr)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// MeasureTPM1 hashes data and extends it into
// a TPM 1.2 PCR your choice.
func MeasureTPM1(pcr uint32, data []byte) error {
	rwc, err := tspi.OpenTPM(tpmDefaultDevice)
	if err != nil {
		return err
	}

	defer rwc.Close()

	hash := sha1.Sum(data)

	if _, err := tspi.PcrExtend(rwc, pcr, hash); err != nil {
		return err
	}

	return nil
}
