package bootconfig

// BootConfig is a boot configuration
// Name: Unique identifier
// DeviceTree: Filename of the device-tree
// Kernel: Filename of the kernel
// Initrd: Filename of the initrd
// CommandLine: Kernel commandline
type BootConfig struct {
	Name        string `json:"name"`
	DeviceTree  string `json:"dt"`
	Kernel      string `json:"kernel"`
	Initrd      string `json:"initrd"`
	CommandLine string `json:"commandline"`
}

// ManifestConfig is the boot manifest containing multiple boot configurations
type ManifestConfig struct {
	Configs []BootConfig `json:"configs"`
}
