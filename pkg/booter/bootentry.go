package booter

import (
	"fmt"
	"log"

	"github.com/insomniacslk/systemboot/pkg/vpd"
)

// Get, Set and GetAll are defined here as variables so they can be overridden
// for testing, or for using a key-value store other than VPD.
var (
	Get    = vpd.Get
	Set    = vpd.Set
	GetAll = vpd.GetAll
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
	NewVerifiedBooter,
}

// GetBooterFor looks for a supported Booter implementation and returns it, if
// found. If not found, a NullBooter is returned.
func GetBooterFor(entry BootEntry) Booter {
	var (
		booter Booter
		err    error
	)
	for idx, booterParser := range supportedBooterParsers {
		log.Printf("Trying booter #%d", idx)
		booter, err = booterParser(entry.Config)
		if err != nil {
			log.Printf("This config is not valid for this booter (#%d)", idx)
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

func bootEntryExistsAndRemove(name string, entries []BootEntry) []BootEntry {
	for i := 0; i < len(entries); i++ {
		if name == entries[i].Name {
			entries[i] = entries[len(entries)-1]
			return entries[:len(entries)-1]
		}
	}
	return entries
}

// GetBootEntries returns a list of BootEntry objects stored in the VPD
// partition of the flash chip
// Fallback via Boot9999->Boot0000 and overwrites for RO variable store.
func GetBootEntries() []BootEntry {
	var bootEntries []BootEntry
	for idx := 9999; idx >= 0; idx-- {
		key := fmt.Sprintf("Boot%04d", idx)
		// try the RW entries first
		value, err := Get(key, false)
		if err == nil {
			bootEntries = append(bootEntries, BootEntry{Name: key, Config: value})
		}

		// try the RO entries first
		value, err = Get(key, true)
		if err == nil {
			// Check for duplication
			bootEntries = bootEntryExistsAndRemove(key, bootEntries)
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
