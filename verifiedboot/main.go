package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/systemboot/systemboot/pkg/bootconfig"
	"github.com/systemboot/systemboot/pkg/recovery"
	"github.com/systemboot/systemboot/pkg/rng"
	"github.com/systemboot/systemboot/pkg/storage"
	"github.com/systemboot/systemboot/pkg/tpm"
)

const (
	// Version of verified booter
	Version = `0.1`
	// LinuxPcrIndex for Linux measurements
	LinuxPcrIndex = 7
	// LinuxDevUUIDPath sysfs path
	LinuxDevUUIDPath = "/dev/disk/by-uuid/"
	// BaseMountPoint is the basic mountpoint Path
	BaseMountPoint = "/mnt/"
)

// SignaturePublicKeyPath is the public key path for signature verifcation
var SignaturePublicKeyPath = "/etc/security/public_key.pem"

var banner = `

[0m[0m ________________________________________________________ [0m
[0m/ I simply cannot let such a crime against fabulosity go[0m \[0m
[0m\ uncorrected! Verified booter v` + Version + `[0m                      /[0m
[0m -------------------------------------------------------- [0m[00m
     [0m\[0m                                               [00m
      [0m\[0m                                              [00m
       [0m\[0m [38;5;188m▄▄[48;5;54;38;5;97m▄▄[49;38;5;54m▄[48;5;54m█[38;5;97m▄▄▄[49;38;5;54m▄▄[39m                                 [00m
       [38;5;54m▄[48;5;54m█[48;5;188m▄[48;5;255;38;5;254m▄[48;5;188;38;5;255m▄[48;5;97;38;5;188m▄[38;5;97m█[48;5;54m▄[38;5;54m█[48;5;97;38;5;97m████[48;5;188;38;5;188m█[38;5;255m▄▄[49;38;5;188m▄[39m                             [00m
     [38;5;54m▄[48;5;54;38;5;97m▄[48;5;97m███[48;5;54m▄[48;5;254;38;5;254m█[48;5;255;38;5;255m█[48;5;188m▄[38;5;188m█[48;5;54;38;5;255m▄▄▄▄▄[48;5;188;38;5;188m█[48;5;255;38;5;255m█[38;5;254m▄[48;5;188;38;5;255m▄[49;38;5;188m▄[39m                            [00m
    [38;5;54m▄[48;5;97m▄▄[48;5;54m████[38;5;188m▄[48;5;254;38;5;255m▄▄[48;5;255m████████[48;5;188;38;5;188m█[48;5;255;38;5;255m█[48;5;188;38;5;188m█[49;39m                            [00m
    [48;5;54;38;5;54m████[38;5;97m▄[38;5;54m█[48;5;97;38;5;188m▄[48;5;188;38;5;255m▄[48;5;255m██[38;5;117m▄[48;5;117;38;5;16m▄▄▄[48;5;188m▄[48;5;255m▄[38;5;188m▄[38;5;255m███[48;5;188;38;5;97m▄[49;38;5;54m▄[39m                           [00m
      [38;5;54m▀[48;5;97m▄[38;5;97m█[38;5;133m▄[48;5;188;38;5;188m█[48;5;255;38;5;255m██[38;5;117m▄[48;5;16;38;5;16m█[48;5;68;38;5;231m▄[38;5;68m█[48;5;231;38;5;231m██[48;5;188;38;5;16m▄[48;5;255m▄[38;5;255m██[48;5;188;38;5;188m█[48;5;97;38;5;97m█[48;5;54;38;5;54m█[49;39m                           [00m
     [38;5;54m▄▄[48;5;54;38;5;97m▄[38;5;54m█[48;5;133;38;5;188m▄[48;5;188m█[48;5;255;38;5;255m██[48;5;16;38;5;16m█[38;5;231m▄[38;5;16m█[48;5;68;38;5;68m█[48;5;231;38;5;231m██[48;5;188;38;5;16m▄[48;5;255;38;5;255m███[48;5;188;38;5;188m█[48;5;97;38;5;97m█[48;5;54;38;5;54m██[49;39m       [38;5;54m▄▄[48;5;54;38;5;133m▄▄▄▄[38;5;97m▄▄[49;38;5;54m▄▄[39m         [00m
     [48;5;54;38;5;54m█[48;5;133;38;5;133m█[48;5;54;38;5;97m▄[38;5;54m█[48;5;188;38;5;188m█[48;5;255;38;5;255m█[38;5;254m▄[38;5;255m█[48;5;16m▄▄[48;5;68m▄[48;5;231m▄▄[48;5;188m▄[48;5;255m███[48;5;188;38;5;188m█[48;5;97;38;5;97m█[38;5;54m▄[48;5;54;38;5;133m▄[38;5;54m█[49;39m     [38;5;54m▄[48;5;54;38;5;133m▄[48;5;133;38;5;97m▄[48;5;97;38;5;133m▄▄▄[48;5;54m▄[38;5;97m▄▄▄[38;5;54m█[48;5;97m▄[48;5;54;38;5;97m▄[49;38;5;54m▄[39m       [00m
      [38;5;54m▀▀▀▀[38;5;188m▀[48;5;188m█[48;5;255m▄▄▄[38;5;255m███████[48;5;188;38;5;54m▄[48;5;97;38;5;97m█[48;5;54;38;5;54m█[48;5;133;38;5;133m█[48;5;97;38;5;97m█[48;5;54;38;5;54m█[49;39m   [38;5;54m▄[48;5;54;38;5;133m▄[48;5;97;38;5;54m▄[48;5;54;38;5;133m▄▄▄▄[38;5;97m▄[48;5;97;38;5;54m▄▄[48;5;54m███[48;5;97;38;5;97m█[48;5;54m▄[49;38;5;54m▄[39m      [00m
               [38;5;188m▀▀[48;5;188m█[48;5;255;38;5;255m███[38;5;54m▄[48;5;54m█[38;5;97m▄[48;5;133;38;5;54m▄[48;5;97m▄[48;5;133;38;5;133m█[48;5;54;38;5;54m█[49;38;5;188m▄▄▄[48;5;54;38;5;133m▄▄[48;5;133;38;5;54m▄▄[49m▀▀[48;5;97m▄▄▄[48;5;54m████[48;5;97;38;5;97m██[48;5;54;38;5;54m█[49;39m      [00m
                 [48;5;188;38;5;188m█[48;5;255;38;5;255m█[38;5;54m▄[48;5;54m██[48;5;97m▄[48;5;54;38;5;133m▄[38;5;97m▄[38;5;54m█[48;5;133m▄[48;5;54;38;5;255m▄[48;5;255m███[38;5;117m▄[48;5;188;38;5;188m█[49;39m      [48;5;54;38;5;54m███[38;5;97m▄[48;5;97m██[48;5;54;38;5;54m██[49;39m      [00m
                 [48;5;188;38;5;188m█[48;5;255;38;5;255m█[48;5;54m▄[38;5;54m█[38;5;97m▄▄▄[48;5;97;38;5;54m▄[48;5;54;38;5;97m▄[38;5;54m█[48;5;255;38;5;255m██[38;5;75m▄[38;5;255m█[48;5;75m▄[48;5;255m█[48;5;188;38;5;188m█[49;39m    [38;5;54m▄[48;5;54;38;5;97m▄▄[48;5;97m█[38;5;54m▄▄[48;5;54;38;5;97m▄[48;5;97;38;5;54m▄[49m▀[39m      [00m
                  [48;5;188;38;5;188m█[48;5;255;38;5;255m██[48;5;54;38;5;54m█[38;5;97m▄▄[48;5;97m█[38;5;54m▄[48;5;54m█[48;5;255;38;5;188m▄[38;5;255m█[48;5;117m▄[48;5;255m█[48;5;75;38;5;117m▄[48;5;255;38;5;255m█[48;5;188;38;5;188m█[49;39m  [48;5;54;38;5;54m█[38;5;97m▄[48;5;97m██[38;5;54m▄[48;5;54;38;5;97m▄[48;5;97m██[38;5;54m▄[49m▀[39m       [00m
                   [48;5;188;38;5;250m▄[48;5;255;38;5;188m▄[38;5;255m█[48;5;54m▄▄▄[48;5;255m██[48;5;188m▄[48;5;255;38;5;188m▄[38;5;255m███[48;5;188;38;5;188m█[49;39m    [48;5;54;38;5;54m█[48;5;97m▄[48;5;54;38;5;97m▄[48;5;97m███[38;5;54m▄[48;5;54m█[38;5;97m▄[38;5;54m█[49;39m      [00m
                   [48;5;250;38;5;250m█[38;5;254m▄[48;5;188;38;5;188m█[48;5;255;38;5;255m██[48;5;188;38;5;188m█[49m▀▀[48;5;188;38;5;250m▄[38;5;254m▄[38;5;250m▄[48;5;255;38;5;188m▄[38;5;255m█[48;5;188m▄▄[49;38;5;188m▄[39m   [48;5;54;38;5;54m█[48;5;97;38;5;97m██[38;5;54m▄[48;5;54m██[38;5;97m▄[48;5;97m█[48;5;54;38;5;54m██[49;39m [48;5;54;38;5;54m██[49m▄[39m [00m
                   [48;5;250;38;5;250m█[48;5;254;38;5;254m█[48;5;188;38;5;188m█[48;5;255;38;5;255m██[48;5;188;38;5;188m█[49;39m   [48;5;250;38;5;250m█[48;5;254;38;5;254m█[48;5;188m▄[48;5;255;38;5;188m▄[38;5;255m██[48;5;188;38;5;188m█[49;39m   [38;5;54m▀▀▀[48;5;54m██[38;5;97m▄[48;5;97m█[38;5;54m▄[48;5;54;38;5;97m▄[38;5;54m█[49m▄[48;5;54m█[38;5;97m▄[48;5;97;38;5;54m▄[49m▀[39m[00m
                  [48;5;250;38;5;250m█[48;5;254;38;5;254m█[48;5;188;38;5;188m█[48;5;255;38;5;255m███[48;5;188;38;5;188m█[49;39m   [48;5;250;38;5;250m█[48;5;254;38;5;254m██[48;5;188;38;5;188m█[48;5;255;38;5;255m██[48;5;188m▄[49;38;5;188m▄[39m     [38;5;54m▀▀▀[48;5;97m▄[48;5;54;38;5;97m▄[48;5;97m█[48;5;54;38;5;54m█[49m▀[48;5;97m▄▄[49m▀[39m [00m
                 [38;5;250m▄[48;5;250;38;5;254m▄[48;5;254m█[48;5;188;38;5;188m█[48;5;255;38;5;255m███[48;5;188;38;5;188m█[49;39m   [48;5;250;38;5;250m█[48;5;254;38;5;254m██[48;5;188;38;5;188m█[48;5;255;38;5;255m███[48;5;188;38;5;188m█[49;39m         [38;5;54m▀▀▀[39m     [00m
                [48;5;250;38;5;250m█[38;5;254m▄[48;5;254m█[48;5;188;38;5;188m█[48;5;255;38;5;255m████[48;5;188;38;5;188m█[49;39m   [48;5;250;38;5;250m█[48;5;254;38;5;254m██[48;5;188;38;5;188m█[48;5;255;38;5;255m███[48;5;188m▄[49;38;5;188m▄[39m                [00m
                [38;5;250m▀▀[48;5;188;38;5;188m█[38;5;255m▄[48;5;255m███[38;5;188m▄[49m▀[39m   [38;5;250m▀▀▀[48;5;188;38;5;188m█[48;5;255;38;5;255m████[48;5;188;38;5;188m█[49;39m                [00m
                  [38;5;188m▀▀▀▀▀▀[39m       [38;5;188m▀▀▀▀▀▀[39m                [00m
                                                     [00m

`

var (
	doDebug            = flag.Bool("D", true, "Print debug output")
	deviceUUID         = flag.String("d", "", "Block device identified by UUID which should be used")
	bootConfigFilePath = flag.String("b", "", "Boot config image file path on block device")
	bootConfigName     = flag.String("n", "", "Boot config name to use")
	debug              func(string, ...interface{})
	publicKey          []byte
	tpmInterface       tpm.TPM
)

func kexec(kernelFilePath string, kernelCommandline string, initrdFilePath string, dtFilePath string) error {
	var err error
	var baseCmd string
	switch runtime.GOARCH {
	case "AMD64":
		baseCmd, err = exec.LookPath("kexec-amd64")
		if err != nil {
			return err
		}
	case "ARM64":
		baseCmd, err = exec.LookPath("kexec-arm64")
		if err != nil {
			return err
		}
	default:
		return errors.New("Platform for kexec not supported")
	}

	var loadCommands []string
	loadCommands = append(loadCommands, "-l")
	loadCommands = append(loadCommands, kernelFilePath)

	if kernelCommandline != "" {
		loadCommands = append(loadCommands, "--command-line="+kernelCommandline)
	} else {
		loadCommands = append(loadCommands, "--reuse-cmdline")
	}

	if initrdFilePath != "" {
		loadCommands = append(loadCommands, "--initrd="+initrdFilePath)
	}

	cmdLoad := exec.Command(baseCmd, loadCommands...)
	if err := cmdLoad.Run(); err != nil {
		return err
	}

	// Execute into new kernel
	cmdExec := exec.Command(baseCmd, "-e")
	return cmdExec.Run()
}

func main() {
	flag.Parse()
	log.Print(banner)

	var RecoveryHandler recovery.Recoverer
	debug = func(string, ...interface{}) {}
	if *doDebug {
		debug = log.Printf
	}

	RecoveryHandler = recovery.SecureRecoverer{
		Reboot: true,
		Sync:   false,
		Debug:  *doDebug,
	}

	// Initialize random seeding
	if err := rng.UpdateLinuxRandomness(RecoveryHandler); err != nil {
		RecoveryHandler.Recover("Can't setup randomness seeder: " + err.Error())
	}

	// Initialize the TPM
	tpmInterface, err := tpm.NewTPM()
	if err != nil {
		RecoveryHandler.Recover("Can't setup TPM connection: " + err.Error())
	}

	if err = tpmInterface.SetupTPM(); err != nil {
		RecoveryHandler.Recover("Can't setup TPM state machine: " + err.Error())
	}

	// Check if device by UUID exists
	devicePath := path.Join(LinuxDevUUIDPath, *deviceUUID)
	if _, err = os.Stat(devicePath); err != nil {
		RecoveryHandler.Recover("Can't find device by UUID: " + err.Error())
	}

	// Check supported filesystems
	filesystems, err := storage.GetSupportedFilesystems()
	if err != nil {
		RecoveryHandler.Recover("Can't read supported filesystems: " + err.Error())
	}

	// Mount device under base path
	mountPath := path.Join(BaseMountPoint, *deviceUUID)
	mountpoint, err := storage.Mount(devicePath, mountPath, filesystems)
	if err != nil {
		RecoveryHandler.Recover("Can't mount device " + devicePath + " under path " + mountPath + " because of error: " + err.Error())
	}

	// Check FIT image existence and read it into memory
	bootConfigImage := path.Join(mountpoint.Path, *bootConfigFilePath)
	bootConfigImageData, err := ioutil.ReadFile(bootConfigImage)
	if err != nil {
		RecoveryHandler.Recover("Can't read boot config image by given path: " + err.Error())
	}

	bc, artifacts, err := bootconfig.GetBootConfig(*bootConfigFilePath, &SignaturePublicKeyPath, *bootConfigName)
	if err != nil {
		RecoveryHandler.Recover("Can't unpack boot config image by given path: " + err.Error())
	}

	// Measure FIT image into linux PCR
	err = tpmInterface.Measure(LinuxPcrIndex, bootConfigImageData)
	if err != nil {
		RecoveryHandler.Recover("Can't measure boot config image hash and extend it into the TPM: " + err.Error())
	}

	kernelFilePath := path.Join(*artifacts, bootconfig.DefaultKernelPath, bc.Kernel)
	initrdFilePath := path.Join(*artifacts, bootconfig.DefaultInitrdPath, bc.Initrd)
	dtFilePath := path.Join(*artifacts, bootconfig.DefaultDeviceTreePath, bc.DeviceTree)

	err = kexec(kernelFilePath, bc.CommandLine, initrdFilePath, dtFilePath)
	if err != nil {
		RecoveryHandler.Recover("Can't kexec into new kernel: " + err.Error())
	}
}
