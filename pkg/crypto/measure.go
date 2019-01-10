package crypto

import (
	"io/ioutil"
	"log"

	"github.com/systemboot/tpmtool/pkg/tpm"
)

const (
	// BlobPCR type in PCR 7
	BlobPCR uint32 = 7
	// BootConfigPCR type in PCR 8
	BootConfigPCR uint32 = 8
	// ConfigDataPCR type in PCR 8
	ConfigDataPCR uint32 = 8
	// NvramVarsPCR type in PCR 9
	NvramVarsPCR uint32 = 9
)

// TryMeasureBootConfig measures bootconfig contents
func TryMeasureBootConfig(name, kernel, initramfs, kernelArgs, deviceTree, multiboot, multibootArgs string, modules []string) {
	TPMInterface, err := tpm.NewTPM()
	if err != nil {
		log.Printf("Cannot open TPM: %v", err)
		return
	}
	TryMeasureData(BootConfigPCR, []byte(name), name)
	TryMeasureData(BootConfigPCR, []byte(kernel), kernel)
	TryMeasureData(BootConfigPCR, []byte(initramfs), initramfs)
	TryMeasureData(BootConfigPCR, []byte(kernelArgs), kernelArgs)
	TryMeasureData(BootConfigPCR, []byte(deviceTree), deviceTree)
	TryMeasureData(BootConfigPCR, []byte(multiboot), multiboot)
	TryMeasureData(BootConfigPCR, []byte(multibootArgs), multibootArgs)
	for i, module := range modules {
		TryMeasureData(BootConfigPCR, []byte(module), module+string(i))
	}
	TryMeasureFiles(kernel, initramfs, deviceTree, multiboot)
	TPMInterface.Close()
}

// TryMeasureData measures a byte array with additional information
func TryMeasureData(pcr uint32, data []byte, info string) {
	TPMInterface, err := tpm.NewTPM()
	if err != nil {
		log.Printf("Cannot open TPM: %v", err)
		return
	}
	log.Printf("Measuring blob: %v", info)
	TPMInterface.Measure(pcr, data)
	TPMInterface.Close()
}

// TryMeasureFiles measures a variable amount of files
func TryMeasureFiles(files ...string) {
	TPMInterface, err := tpm.NewTPM()
	if err != nil {
		log.Printf("Cannot open TPM: %v", err)
		return
	}
	for _, file := range files {
		log.Printf("Measuring file: %v", file)
		data, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}
		TPMInterface.Measure(BlobPCR, data)
	}
	TPMInterface.Close()
}
