package recovery

import (
	"log"
	"os"
	"syscall"
	"time"
)

const debugTimeout time.Duration = 10

// SecureRecoverer properties
// Reboot: does a reboot if true
// Sync: sync file descriptors and devices
type SecureRecoverer struct {
	Reboot bool
	Sync   bool
	Debug  bool
}

// Recover by reboot or poweroff without or with sync
func (sr SecureRecoverer) Recover(message string) error {
	var err = nil
	if sr.Sync {
		for _, f := range []*os.File{
			os.Stdout,
			os.Stderr,
		} {
			err = f.Sync()
		}
		err = syscall.Sync()
	}

	if sr.Debug {
		if message != "" {
			log.Printf("%s\n", message)
		}
		time.Sleep(debugTimeout * time.Second)
	}

	if sr.Reboot {
		err = syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	} else {
		err = syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	}

	return err
}
