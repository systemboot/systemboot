package booter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	DevicePath string `json:"device_path"`
	BCFile     string `json:"bc_file"`
	BCName     string `json:"bc_name"`
}

// NewVerifiedBooter parses a boot entry config and returns a Booter instance, // or an error if any
func NewVerifiedBooter(config []byte) (Booter, error) {
	// The configuration format for a VerifiedBooter entry is a JSON with the
	// following structure:
	// {
	//     "type": "verifiedboot",
	//     "device_path": "<path>",
	//     "bc_file": "<path>",
	//     "bc_name": "<string>",
	// }
	//
	// `type` is always set to "verifiedboot".
	// `device_path` is the path of the block device which contains the fit_file.
	// `boot_config` is an absolute filepath containing a fit image.
	//
	// An example configuration is:
	// {
	//     "type": "verified",
	//     "device_path": "/dev/sda1",
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

	if nb.DevicePath == "" || !filepath.IsAbs(nb.DevicePath) {
		return nil, fmt.Errorf("Device file path is incorrect for VerifiedBooter %s", nb.DevicePath)
	}

	if nb.BCFile == "" || !filepath.IsAbs(nb.BCFile) {
		return nil, fmt.Errorf("BootConfig file path is incorrect for VerifiedBooter")
	}

	return &nb, nil
}

// Boot will run the boot procedure. In the case of VerifiedBooter, it will
// call the `verifiedboot` command
func (nb *VerifiedBooter) Boot() error {
	bootcmd := []string{"verifiedboot", "-d", nb.DevicePath, "-b", nb.BCFile, "-n", nb.BCName}

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
