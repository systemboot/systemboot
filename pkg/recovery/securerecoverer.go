package recovery

import (
	"os"
	"syscall"
)

// SecureRecoverer properties
// Reboot: does a reboot if true
// Sync: sync file descriptors and devices
type SecureRecoverer struct {
	Reboot bool
	Sync   bool
}

// Recover by reboot or poweroff without or with sync
func (sr SecureRecoverer) Recover() error {
	if sr.Sync {
		for _, f := range []*os.File{
			os.Stdout,
			os.Stderr,
		} {
			syscall.Fsync(int(f.Fd()))
		}
		syscall.Sync()
	}

	if sr.Reboot {
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	} else {
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	}

	return nil
}
