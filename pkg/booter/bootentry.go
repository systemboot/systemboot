package booter

import (
	"fmt"
	"log"

	"github.com/insomniacslk/systemboot/pkg/vpd"
)

// BootEntry represents a boot entry, with its name, configuration, and Booter
// instance. It can map to existing key-value stores like VPD or EFI vars.
type BootEntry struct {
	Name   string
	Config []byte
	Booter Booter
}

var supportedBooterParsers = []func([]byte) (Booter, error){
	NewNetBooter,
}

// GetBooterFor looks for a supported Booter implementation and returns it, if
// found. If not found, a NullBooter is returned.
func GetBooterFor(entry BootEntry) Booter {
	var (
		booter Booter
		err    error
	)
	for _, booterParser := range supportedBooterParsers {
		log.Printf("Trying booter: %+v", booterParser)
		booter, err = booterParser(entry.Config)
		if err != nil {
			log.Printf("This config is not valid for booter %+v", booterParser)
			continue
		}
		break
	}
	if booter == nil {
		log.Printf("No booter found for entry: %+v", entry)
		return &NullBooter{}
	}
	return booter
}

// GetBootEntries returns a list of BootEntry objecsts stored in the VPD
// partition of the flash chip
func GetBootEntries() []BootEntry {
	var bootEntries []BootEntry
	for idx := 0; idx < 9999; idx++ {
		key := fmt.Sprintf("Boot%04d", idx)
		// try the RW entries first
		value, err := vpd.Get(key, false)
		if err == nil {
			bootEntries = append(bootEntries, BootEntry{Name: key, Config: value})
			// WARNING WARNING WARNING this means that read-write boot entries
			// have priority over read-only ones
			continue
		}
		// try the RO entries then
		value, err = vpd.Get(key, true)
		if err == nil {
			bootEntries = append(bootEntries, BootEntry{Name: key, Config: value})
		}
	}
	// look for a Booter that supports the given configuration
	for idx, entry := range bootEntries {
		entry.Booter = GetBooterFor(entry)
		if entry.Booter == nil {
			log.Printf("No booter found for entry: %+v", entry)
		}
		bootEntries[idx] = entry
	}
	return bootEntries
}
