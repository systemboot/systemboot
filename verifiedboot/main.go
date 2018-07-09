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
	Version = `0.2`
	// LinuxPcrIndex for Linux measurements
	LinuxPcrIndex = 9
	// BaseMountPoint is the basic mountpoint Path
	BaseMountPoint = "/mnt"
	// FirstDTBPath is the first dtp path to check
	FirstDTBPath = "/sys/firmware/fdt"
	// SecondDTBPath is the second dtp path to check
	SecondDTBPath = "/proc/device-tree"
)

// SignaturePublicKeyPath is the public key path for signature verifcation
var SignaturePublicKeyPath = "/etc/security/key.pem"

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
	devicePath         = flag.String("d", "", "Block device identified by path which should be used")
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
	case "amd64":
		baseCmd, err = exec.LookPath("kexec-amd64")
		if err != nil {
			return err
		}
	case "arm64":
		baseCmd, err = exec.LookPath("kexec-arm64")
		if err != nil {
			return err
		}
	default:
		return errors.New("Platform has no kexec tool support")
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

	if dtFilePath != "" {
		loadCommands = append(loadCommands, "--dtb="+dtFilePath)
	} else {
		_, err := os.Stat(FirstDTBPath)
		if err == nil {
			loadCommands = append(loadCommands, "--dtb="+FirstDTBPath)
		} else {
			_, err := os.Stat(SecondDTBPath)
			if err == nil {
				loadCommands = append(loadCommands, "--dtb="+SecondDTBPath)
			}
		}
	}

	// Load data into physical non reserved memory regions
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
		Sync:   true,
		Debug:  *doDebug,
	}

	// Initialize random seeding
	if err := rng.UpdateLinuxRandomness(RecoveryHandler); err != nil {
		log.Printf("Couldn't initialize random seeder: %s\n", err.Error())
	}

	// Initialize the TPM
	tpmInterface, err := tpm.NewTPM()
	if err != nil {
		RecoveryHandler.Recover("Can't setup TPM connection: " + err.Error())
	}

	if err = tpmInterface.SetupTPM(); err != nil {
		RecoveryHandler.Recover("Can't setup TPM state machine: " + err.Error())
	}

	// Check supported filesystems
	filesystems, err := storage.GetSupportedFilesystems()
	if err != nil {
		RecoveryHandler.Recover("Can't read supported filesystems: " + err.Error())
	}

	os.MkdirAll(BaseMountPoint, 1755)

	// Mount device under base path
	mountPath, err := ioutil.TempDir(BaseMountPoint, "")
	if err != nil {
		RecoveryHandler.Recover("Can't create temporary mount path: " + err.Error())
	}

	mountpoint, err := storage.Mount(*devicePath, mountPath, filesystems)
	if err != nil {
		RecoveryHandler.Recover("Can't mount device " + *devicePath + " under path " + mountPath + " because of error: " + err.Error())
	}
	bootConfigImagePath := path.Join(mountpoint.Path, *bootConfigFilePath)

	// Check boot config image existence and read it into memory
	bootConfigImageData, err := ioutil.ReadFile(bootConfigImagePath)
	if err != nil {
		RecoveryHandler.Recover("Can't read boot config image by given path: " + err.Error())
	}

	bc, _, err := bootconfig.GetBootConfig(bootConfigImagePath, &SignaturePublicKeyPath, *bootConfigName)
	if err != nil {
		RecoveryHandler.Recover("Can't unpack boot config image by given path: " + err.Error())
	}

	// Measure boot config image into linux PCR
	err = tpmInterface.Measure(LinuxPcrIndex, bootConfigImageData)
	if err != nil {
		RecoveryHandler.Recover("Can't measure boot config image hash and extend it into the TPM: " + err.Error())
	}

	err = kexec(bc.Kernel, bc.CommandLine, bc.Initrd, bc.DeviceTree)
	if err != nil {
		RecoveryHandler.Recover("Can't kexec into new kernel: " + err.Error())
	}
}
