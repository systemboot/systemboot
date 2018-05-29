package recovery

import (
	"os"
	"syscall"
)

// PowerAction gives poweoff properties
type PowerAction struct {
	reboot bool
	sync   bool
}

// PowerCycle implements linux based shutdown
// behaviour
func (a PowerAction) PowerCycle() {
	if a.sync {
		for _, f := range []*os.File{
			os.Stdout,
			os.Stderr,
		} {
			syscall.Fsync(int(f.Fd()))
		}
		syscall.Sync()
	}

	if a.reboot {
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	} else {
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	}
}
