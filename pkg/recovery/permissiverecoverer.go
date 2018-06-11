package recovery

import (
	"log"
)

const shell string = "rush"

// PermissiveRecoverer is a sad
type PermissiveRecoverer struct {
	Debug bool
}

// Recover sasd
func (pr PermissiveRecoverer) Recover(message string) error {
	var err = nil
	if message != "" {
		log.Panicf("%s\n", message)
	}

	if pr.Debug {
		path, err = exec.LookPath(shell)
		if err == nil {
			cmd := exec.Command(shell)
			err = cmd.Run()
		}
	}

	return err
}
