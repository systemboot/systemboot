package booter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// StBooter implements the Booter interface for booting system transpatrency systems.
type StBooter struct {
	Type string `json:"type"`
}

// NewStBooter parses a bootentry config and returns a Booter instance, or
// an error if any
func NewStBooter(config []byte) (Booter, error) {
	/*
		The configuration format for a StBooter is a JSON with the following structure:

		{
			"type": "stboot"
		}

		`type` is always set to "stboot"

		Currently there are no further information needed for the StBooter to work.
	*/
	log.Printf("Trying StBooter...")
	log.Printf("Config: %s", string(config))
	stb := StBooter{}
	if err := json.Unmarshal(config, &stb); err != nil {
		return nil, err
	}
	log.Printf("StBooter: %+v", stb)
	if stb.Type != "stboot" {
		return nil, fmt.Errorf("Wrong type for StBooter: %s", stb.Type)
	}
	// the actual arguments validation is done in `Boot` to avoid duplicate code
	return &stb, nil
}

// Boot will run the boot procedure. In the case of StBooter, it will call
// the `stboot` command
func (stb *StBooter) Boot() error {
	bootcmd := []string{"stboot", "-d"}
	// currently there are no more arguments

	log.Printf("Executing command: %v", bootcmd)
	cmd := exec.Command(bootcmd[0], bootcmd[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing %v: %v", cmd, err)
	}
	return nil
}

// TypeName returns the name of the booter type
func (stb *StBooter) TypeName() string {
	return stb.Type
}
