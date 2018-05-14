package security

import (
	"crypto/sha1"

	tspi "github.com/google/go-tpm/tpm"
)

const (
	TpmDefaultDevice     string = "/dev/tpm0"
	Tpm1DeactivatedError        = 0x00000006
	Tpm1DisabledError           = 0x00000007
)

// OwnerClearTPM1 clears the TPM and destorys all
// access to existing keys. Afterwards a machine
// power cycle is needed.
func OwnerClearTPM1(ownerPassword string) error {
	var ownerAuth [20]byte
	rwc, err := tspi.OpenTPM(TpmDefaultDevice)
	if err != nil {
		return err
	}

	defer rwc.Close()

	if ownerPassword != "" {
		ownerAuth = sha1.Sum([]byte(ownerPassword))
	}

	return tspi.OwnerClear(rwc, ownerAuth)
}

// TakeOwnershipTPM1 takes ownership of the TPM. if no password defined use
// WELL_KNOWN_SECRET aka 20 zero bytes.
func TakeOwnershipTPM1(ownerPassword string, srkPassword string) error {
	var ownerAuth [20]byte
	var srkAuth [20]byte
	rwc, err := tspi.OpenTPM(TpmDefaultDevice)
	if err != nil {
		return err
	}

	defer rwc.Close()

	if ownerPassword != "" {
		ownerAuth = sha1.Sum([]byte(ownerPassword))
	}

	if srkPassword != "" {
		srkAuth = sha1.Sum([]byte(srkPassword))
	}

	// This test assumes that the TPM has been cleared using OwnerClear.
	pubek, err := tspi.ReadPubEK(rwc)
	if err != nil {
		return err
	}

	return tspi.TakeOwnership(rwc, ownerAuth, srkAuth, pubek)
}

// ReadPcrTPM1 reads the PCR for the given
// index
func ReadPcrTPM1(pcr uint32) ([]byte, error) {
	rwc, err := tspi.OpenTPM(TpmDefaultDevice)
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
	rwc, err := tspi.OpenTPM(TpmDefaultDevice)
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
