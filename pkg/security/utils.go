package security

import (
	"log"
	"syscall"
	"time"
)

// Die can be used to hard power off
// or reboot a system in case of an error
// security violation.
func Die(msg string, reboot bool, timeout time.Duration) {
	log.Fatal(msg)

	time.Sleep(timeout * time.Second)

	if reboot {
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	} else {
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
	}
}
