package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/systemboot/systemboot/pkg/bootconfig"
)

// List of directories where to recursively look for grub config files. The root dorectory
// of each mountpoint, these folders inside the mountpoint and all subfolders
// of these folders are searched
var (
	GrubSearchDirectories = []string{
		"boot",
		"EFI",
		"efi",
		"grub",
		"grub2",
	}
)

type grubVersion int

var (
	grubV1 grubVersion = 1
	grubV2 grubVersion = 2
)

func isGrubSearchDir(dirname string) bool {
	for _, dir := range GrubSearchDirectories {
		if dirname == dir {
			return true
		}
	}
	return false
}

// ParseGrubCfg parses the content of a grub.cfg and returns a list of
// BootConfig structures, one for each menuentry, in the same order as they
// appear in grub.cfg. All opened kernel and initrd files are relative to
// basedir.
func ParseGrubCfg(ver grubVersion, grubcfg string, basedir string) []bootconfig.BootConfig {
	// This parser sucks. It's not even a parser, it just looks for lines
	// starting with menuentry, linux or initrd.
	// TODO use a parser, e.g. https://github.com/alecthomas/participle
	if ver != grubV1 && ver != grubV2 {
		log.Printf("Warning: invalid GRUB version: %d", ver)
		return nil
	}
	bootconfigs := make([]bootconfig.BootConfig, 0)
	inMenuEntry := false
	var cfg *bootconfig.BootConfig
	for _, line := range strings.Split(grubcfg, "\n") {
		// remove all leading spaces as they are not relevant for the config
		// line
		line = strings.TrimLeft(line, " ")
		sline := strings.Fields(line)
		if len(sline) == 0 {
			continue
		}
		if sline[0] == "menuentry" {
			// if a "menuentry", start a new boot config
			if cfg != nil {
				// save the previous boot config, if any
				if cfg.IsValid() {
					// only consider valid boot configs, i.e. the ones that have
					// both kernel and initramfs
					bootconfigs = append(bootconfigs, *cfg)
				}
			}
			inMenuEntry = true
			cfg = new(bootconfig.BootConfig)
			name := ""
			if len(sline) > 1 {
				name = strings.Join(sline[1:], " ")
				name = unquote(ver, name)
				name = strings.Split(name, "--")[0]
			}
			cfg.Name = name
		} else if inMenuEntry {
			// otherwise look for kernel and initramfs configuration
			if len(sline) < 2 {
				// surely not a valid linux or initrd directive, skip it
				continue
			}
			if sline[0] == "linux" || sline[0] == "linux16" || sline[0] == "linuxefi" {
				kernel := sline[1]
				cmdline := strings.Join(sline[2:], " ")
				cmdline = unquote(ver, cmdline)
				cfg.Kernel = path.Join(basedir, kernel)
				cfg.KernelArgs = cmdline
			} else if sline[0] == "initrd" || sline[0] == "initrd16" || sline[0] == "initrdefi" {
				initrd := sline[1]
				cfg.Initramfs = path.Join(basedir, initrd)
			} else if sline[0] == "multiboot" || sline[0] == "multiboot2" {
				multiboot := sline[1]
				cmdline := strings.Join(sline[2:], " ")
				cmdline = unquote(ver, cmdline)
				cfg.Multiboot = path.Join(basedir, multiboot)
				cfg.MultibootArgs = cmdline
			} else if sline[0] == "module" || sline[0] == "module2" {
				module := sline[1]
				cmdline := strings.Join(sline[2:], " ")
				cmdline = unquote(ver, cmdline)
				module = path.Join(basedir, module)
				if cmdline != "" {
					module = module + " " + cmdline
				}
				cfg.Modules = append(cfg.Modules, module)
			}
		}
	}
	// append last kernel config if it wasn't already
	if inMenuEntry && cfg.IsValid() {
		bootconfigs = append(bootconfigs, *cfg)
	}
	return bootconfigs
}

func unquote(ver grubVersion, text string) string {
	if ver == grubV2 {
		// if grub2, unquote the string, as directives could be quoted
		// https://www.gnu.org/software/grub/manual/grub/grub.html#Quoting
		// TODO unquote everything, not just \$
		return strings.Replace(text, `\$`, "$", -1)
	}
	// otherwise return the unmodified string
	return text
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

// ScanGrubConfigs looks for grub2 and grub legacy config files in the known
// locations and returns a list of boot configurations.
func ScanGrubConfigs(basedir string) []bootconfig.BootConfig {
	bootconfigs := make([]bootconfig.BootConfig, 0)
	err := filepath.Walk(basedir, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
		currentPath, _, _ = transform.String(t, currentPath)
		if info.IsDir() {
			if path.Dir(currentPath) == basedir && !isGrubSearchDir(path.Base(currentPath)) {
				debug("Skip %s", currentPath)
				// skip irrelevant toplevel directories
				return filepath.SkipDir
			}
			debug("Check %s", currentPath)
			// continue
			return nil
		}
		cfgname := info.Name()
		var ver grubVersion
		switch cfgname {
		case "grub.cfg":
			ver = grubV1
		case "grub2.cfg":
			ver = grubV2
		default:
			return nil
		}
		log.Printf("Parsing %s", currentPath)
		data, err := ioutil.ReadFile(currentPath)
		if err != nil {
			return err
		}
		cfgs := ParseGrubCfg(ver, string(data), basedir)
		bootconfigs = append(bootconfigs, cfgs...)
		return nil
	})
	if err != nil {
		log.Printf("filepath.Walk error: %v", err)
	}
	return bootconfigs
}
