package booter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/satori/go.uuid"
)

const (
	// BootModeVerified enables verified boot mode
	BootModeVerified = "verified"
	// BootModeMeasured enables measured boot mode
	BootModeMeasured = "measured"
	// BootModeBoth enables verified and measured boot mode
	BootModeBoth = "both"
)

// VerifiedBooter implements the Booter interface for booting securely
// into the operating system. This includes verified and measured boot
// meachanisms.
type VerifiedBooter struct {
	Type       string `json:"type"`
	DeviceUUID string `json:"device_uuid"`
	BCFile     string `json:"bc_file"`
	BCName     string `json:"bc_name"`
}

// NewVerifiedBooter parses a boot entry config and returns a Booter instance, // or an error if any
func NewVerifiedBooter(config []byte) (Booter, error) {
	// The configuration format for a VerifiedBooter entry is a JSON with the
	// following structure:
	// {
	//     "type": "verifiedboot",
	//     "device_uuid": "<uuid>",
	//     "bc_file": "<path>",
	//     "bc_name": "<string>",
	// }
	//
	// `type` is always set to "verifiedboot".
	// `device_uuid` is the UUID of the block device which contains the fit_file.
	// `boot_config` is an absolute filepath containing a fit image.
	//
	// An example configuration is:
	// {
	//     "type": "verified",
	//     "device_uuid": "597ca453-ddb4-499b-8385-aa1383133249",
	//     "boot_config": "/boot/fit.img"
	// }
	//
	// Additional options may be added in the future.
	log.Printf("Trying VerifiedBooter...")
	log.Printf("Config: %s", string(config))
	nb := VerifiedBooter{}
	if err := json.Unmarshal(config, &nb); err != nil {
		return nil, err
	}

	log.Printf("VerifiedBooter: %+v", nb)
	if nb.Type != "verifiedboot" {
		return nil, fmt.Errorf("Wrong type for VerifiedBooter: %s", nb.Type)
	}

	if _, err := uuid.FromString(nb.DeviceUUID); err != nil {
		return nil, fmt.Errorf("Not an UUID for VerifiedBooter: %s", nb.DeviceUUID)
	}

	if nb.BootConfig == "" || !filepath.IsAbs(nb.BootConfig) {
		return nil, fmt.Errorf("BootConfig file path is incorrect for VerifiedBooter")
	}

	return &nb, nil
}

// Boot will run the boot procedure. In the case of VerifiedBooter, it will
// call the `verifiedboot` command
func (nb *VerifiedBooter) Boot() error {
	bootcmd := []string{"verifiedboot", "-d", nb.DeviceUUID, "-b", nb.BootConfig}

	log.Printf("Executing command: %v", bootcmd)
	cmd := exec.Command(bootcmd[0], bootcmd[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing %v: %v", cmd, err)
	}
	// This should be never reached
	return nil
}

// TypeName returns the name of the booter type
func (nb *VerifiedBooter) TypeName() string {
	return nb.Type
}
