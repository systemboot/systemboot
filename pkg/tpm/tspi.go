package tpm

import (
	"bytes"
	"io"
	"os"
	"strconv"

	tspi "github.com/google/go-tpm/tpm"
)

const (
	// TPMDevice main device path for
	// TSS usage
	TPMDevice = "/dev/tpm0"

	// TpmCapabilities for selecting tpm spec
	TpmCapabilities = "/sys/class/tpm/tpm0/caps"

	// TpmOwnershipState contains owner state
	TpmOwnershipState = "/sys/class/tpm/tpm0/owned"

	// TpmActivatedState contains active state
	TpmActivatedState = "/sys/class/tpm/tpm0/active"

	// TpmEnabledState contains enabled state
	TpmEnabledState = "/sys/class/tpm/tpm0/enabled"

	tpm12      = "1.2"
	tpm20      = "2.0"
	specFilter = "TCG version: "
)

var (
	// OwnerPassword is the owner password
	OwnerPassword string

	// SrkPassword is the SRK password
	SrkPassword string

	tpmHandle       *TPM
	wellKnownSecret string
)

// TPM global struct containing runtime information
type TPM struct {
	device        io.ReadWriteCloser
	specification string
	owned         bool
	active        bool
	enabled       bool
}

func init() {
	err := NewTPM()
	if err != nil {
		//Die
	}
}

func getInfo() (string, bool, bool, bool, error) {
	var cap [256]byte
	var owned [0]byte
	var active [0]byte
	var enabled [0]byte

	caps, err := os.Open(TpmCapabilities)
	if err != nil {
		return "", false, false, false, err
	}

	ownedState, err := os.Open(TpmCapabilities)
	if err != nil {
		return "", false, false, false, err
	}

	activeState, err := os.Open(TpmCapabilities)
	if err != nil {
		return "", false, false, false, err
	}

	enabledState, err := os.Open(TpmCapabilities)
	if err != nil {
		return "", false, false, false, err
	}

	caps.Read(cap[:])
	specBytes := bytes.Split(cap[:], []byte(specFilter))
	specBytes = bytes.Split(specBytes[1], []byte("\n"))

	ownedState.Read(owned[:])
	activeState.Read(active[:])
	enabledState.Read(enabled[:])

	caps.Close()
	ownedState.Close()
	activeState.Close()
	enabledState.Close()

	spec := string(specBytes[0])
	ownedBool, _ := strconv.ParseBool(string(owned[:]))
	activeBool, _ := strconv.ParseBool(string(active[:]))
	enabledBool, _ := strconv.ParseBool(string(enabled[:]))

	return spec, ownedBool, activeBool, enabledBool, nil
}

// NewTPM gets a new TPM handle struct with
// io fd and specification string
func NewTPM() error {
	rwc, err := tspi.OpenTPM(TPMDevice)
	if err != nil {
		return err
	}

	// No error checking for spec because of tpm 1.2
	// capability command not being available in deacitvated
	// or disabled state.
	spec, owned, active, enabled, _ := getInfo()

	tpmHandle = &TPM{device: rwc, specification: spec, owned: owned, active: active, enabled: enabled}
	return nil
}

// Close io fd
func Close() {
	tpmHandle.device.Close()
}

// SetupTPM enabled, activates and takes
// the ownership of a TPM if it is not in a good
// state
func SetupTPM() error {
	var err error

	if tpmHandle.owned && tpmHandle.specification == tpm12 {
		_, err = tpmHandle.ReadPubEKTPM1(wellKnownSecret)
		if err != nil {
			ClearOwnership()
		}
	}

	if !tpmHandle.owned {
		err := TakeOwnership()
		if err != nil {
			//Die
		}
	}

	if !tpmHandle.enabled || !tpmHandle.active {
		//utils.Die(true, "Please enable the TPM")
	}

	return err
}

// Measure data into a PCR by index
func Measure(pcr uint32, data []byte) error {
	var err error
	if tpmHandle.specification == tpm12 {
		err = tpmHandle.MeasureTPM1(pcr, data)
	}
	return err
}

// TakeOwnership claims TPM ownership
func TakeOwnership() error {
	var err error
	if tpmHandle.specification == tpm12 {
		err = tpmHandle.TakeOwnershipTPM1(OwnerPassword, SrkPassword)
	}
	return err
}

// ClearOwnership clears ownership of the TPM
func ClearOwnership() error {
	var err error
	if tpmHandle.specification == tpm12 {
		err = tpmHandle.OwnerClearTPM1(OwnerPassword)
	}
	return err
}
