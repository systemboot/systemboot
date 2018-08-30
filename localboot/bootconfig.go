package main

import (
	"github.com/u-root/u-root/pkg/kexecbin"
)

// BootConfig holds information to boot a kernel using kexec
type BootConfig struct {
	KernelFilePath string
	InitrdFilePath string
	Cmdline        string
}

// IsValid returns true if the BootConfig has a valid kernel and initrd entry
func (bc BootConfig) IsValid() bool {
	return bc.KernelFilePath != "" && bc.InitrdFilePath != ""
}

// Boot tries to boot the kernel pointed by the BootConfig option, or returns an
// error if it cannot be booted. The kernel is loaded using kexec
func (bc BootConfig) Boot() error {
	if err := kexecbin.KexecBin(bc.KernelFilePath, bc.Cmdline, bc.InitrdFilePath, ""); err != nil {
		return err
	}
	// this should be never reached
	return nil
}
