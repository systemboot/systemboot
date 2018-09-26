package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"syscall"

	"github.com/systemboot/systemboot/pkg/storage"
)

// TODO backward compatibility for BIOS mode with partition type 0xee
// TODO use a proper parser for grub config (see grub.go)

var (
	baseMountPoint = flag.String("m", "/mnt", "Base mount point where to mount partitions")
	doDryRun       = flag.Bool("dryrun", false, "Do not actually kexec into the boot config")
	doDebug        = flag.Bool("d", false, "Print debug output")
	doGrub         = flag.Bool("grub", false, "Use GRUB mode, i.e. look for valid Grub/Grub2 configuration in default locations to boot a kernel. GRUB mode ignores -kernel/-initramfs/-cmdline")
	kernelPath     = flag.String("kernel", "", "Specify the path of the kernel to execute. If using -grub, this argument is ignored")
	initramfsPath  = flag.String("initramfs", "", "Specify the path of the initramfs to load. If using -grub, this argument is ignored")
	kernelCmdline  = flag.String("cmdline", "", "Specify the kernel command line. If using -grub, this argument is ignored")
	deviceGUID     = flag.String("guid", "", "GUID of the device where the kernel (and optionally initramfs) are located. Ignored if -grub is set or if -kernel is not specified")
)

var debug = func(string, ...interface{}) {}

// BootGrubMode tries to boot a kernel in GRUB mode. GRUB mode means:
// * look for every attached storage device
// * try to mount every device using any of the kernel-supported filesystems
// * look for a GRUB configuration in various well-known locations
// * build a list of valid boot configurations from the found GRUB configuration files
// * try to boot every valid boot configuration until one succeeds
//
// The first parameter, `mountedDevices` is a list of storage.Mountpoint
// representing a mounted storage device.
// The second parameter, `dryrun`, will not boot the found configurations if set
// to true.
func BootGrubMode(devices []storage.BlockDev, baseMountpoint string, dryrun bool) error {
	// get a list of supported file systems for real devices (i.e. skip nodev)
	debug("Getting list of supported filesystems")
	filesystems, err := storage.GetSupportedFilesystems()
	if err != nil {
		log.Fatal(err)
	}
	debug("Supported file systems: %v", filesystems)

	// try mounting all the available devices, with all the supported file
	// systems
	debug("trying to mount all the available block devices with all the supported file system types")
	mounted := make([]storage.Mountpoint, 0)
	for _, dev := range devices {
		devname := path.Join("/dev", dev.Name)
		mountpath := path.Join(baseMountpoint, dev.Name)
		if mountpoint, err := storage.Mount(devname, mountpath, filesystems); err != nil {
			debug("Failed to mount %s on %s: %v", devname, mountpath, err)
		} else {
			mounted = append(mounted, *mountpoint)
		}
	}
	log.Printf("mounted: %+v", mounted)

	// search for a valid grub config and extracts the boot configuration
	bootconfigs := make([]BootConfig, 0)
	for _, mountpoint := range mounted {
		bootconfigs = append(bootconfigs, ScanGrubConfigs(mountpoint.Path)...)
	}
	log.Printf("Found %d boot configs", len(bootconfigs))
	for _, cfg := range bootconfigs {
		debug("%+v", cfg)
	}

	// try to kexec into every boot config kernel until one succeeds
	for _, cfg := range bootconfigs {
		debug("Trying boot configuration %+v", cfg)
		if dryrun {
			// note: in dry-run mode this loop breaks at the first entry
			log.Printf("Dry-run, will not actually boot")
			break
		} else {
			if err := cfg.Boot(); err != nil {
				log.Printf("Failed to boot kernel %s: %v", cfg.Kernel.Name(), err)
				cfg.Close()
			}
		}
	}
	log.Print("No boot configuration succeeded")
	// clean up
	for _, mountpoint := range mounted {
		syscall.Unmount(mountpoint.Path, syscall.MNT_DETACH)
	}
	return nil
}

// BootPathMode tries to boot a kernel in PATH mode. This means:
// * look for a partition with the given GUID and mount it
// * look for the kernel and initramfs in the provided locations
// * boot the kernel with the provided command line
//
// The first parameter, `devices`, is a list of `storage.BlockDev` objects to
// look for the given GUID. The second parameter, `guid`, is the partition GUID
// to look for. The third, `dryrun`, will not actually boot the found
// configuration if set to true.
func BootPathMode(devices []storage.BlockDev, baseMountpoint string, guid string, dryrun bool) error {
	debug("Getting list of supported filesystems")
	filesystems, err := storage.GetSupportedFilesystems()
	if err != nil {
		log.Fatal(err)
	}
	debug("Supported file systems: %v", filesystems)

	log.Printf("Looking for partition with GUID %s", guid)
	partitions, err := storage.PartitionsByGUID(devices, guid)
	if err != nil || len(partitions) == 0 {
		return fmt.Errorf("Error looking up for partition with GUID %s", guid)
	}
	log.Printf("Partitions with GUID %s: %+v", guid, partitions)
	if len(partitions) > 1 {
		log.Printf("Warning: more than one partition found with the given GUID. Using the first one")
	}
	dev := partitions[0]
	mountpath := path.Join(baseMountpoint, dev.Name)
	devname := path.Join("/dev", dev.Name)
	if _, err := storage.Mount(devname, mountpath, filesystems); err != nil {
		return fmt.Errorf("Cannot mount %s on %s: %v", dev.Name, mountpath, err)
	}
	fullKernelPath := path.Join(mountpath, *kernelPath)
	fullInitramfsPath := path.Join(mountpath, *initramfsPath)
	kernelFD, err := os.Open(fullKernelPath)
	if err != nil {
		return fmt.Errorf("Cannot open kernel %s: %v", fullKernelPath, err)
	}
	initramfsFD, err := os.Open(fullInitramfsPath)
	if err != nil {
		return fmt.Errorf("Cannot open initramfs %s: %v", fullInitramfsPath, err)
	}
	cfg := BootConfig{
		Kernel:    kernelFD,
		Initramfs: initramfsFD,
		Cmdline:   *kernelCmdline,
	}
	debug("Trying boot configuration %+v", cfg)
	if dryrun {
		log.Printf("Dry-run, will not actually boot")
	} else {
		if err := cfg.Boot(); err != nil {
			cfg.Close()
			return fmt.Errorf("Failed to boot kernel %s: %v", cfg.Kernel.Name(), err)
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if *doGrub && *kernelPath != "" {
		log.Fatal("Options -grub and -kernel are mutually exclusive")
	}
	if *doDebug {
		debug = log.Printf
	}

	// Get all the available block devices
	devices, err := storage.GetBlockStats()
	if err != nil {
		log.Fatal(err)
	}
	// print partition info
	if *doDebug {
		for _, dev := range devices {
			log.Printf("Device: %+v", dev)
			table, err := storage.GetGPTTable(dev)
			if err != nil {
				continue
			}
			log.Printf("  Table: %+v", table)
			for _, part := range table.Partitions {
				log.Printf("    Partition: %+v", part)
				if !part.IsEmpty() {
					log.Printf("      UUID: %s", part.Type.String())
				}
			}
		}
	}

	// TODO boot from EFI system partitions. See storage.FilterEFISystemPartitions

	if *doGrub {
		if err := BootGrubMode(devices, *baseMountPoint, *doDryRun); err != nil {
			log.Fatal(err)
		}
	} else if *kernelPath != "" {
		if err := BootPathMode(devices, *baseMountPoint, *deviceGUID, *doDryRun); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("You must specify either -grub or -kernel")
	}
	os.Exit(1)
}
