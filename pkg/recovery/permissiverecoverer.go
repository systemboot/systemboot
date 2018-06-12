package recovery

import (
	"log"
	"os/exec"
)

// PermissiveRecoverer properties
// Debug: Enables recovery shell
type PermissiveRecoverer struct {
	RecoveryCommand string
}

// Recover logs error message in panic mode.
// Can jump into a shell for later debugging.
func (pr PermissiveRecoverer) Recover(message string) error {
	log.Printf("%s\n", message)

	if pr.RecoveryCommand != "" {
		cmd := exec.Command(pr.RecoveryCommand)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
