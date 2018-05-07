// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

// Package fit parses Flattened Image Tree image tree blob files.
package fit

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	"github.com/zaolin/uinit/core/fdt"
)

type Fit struct {
	Debug         bool
	fdt           *fdt.Tree
	Description   string
	AddressCells  uint32
	TimeStamp     time.Time
	DefaultConfig string
	Images        map[string]*Image
	Configs       map[string]*Config
}

type Config struct {
	Description string
	ImageList   []*Image
	BaseAddr    uint64
	NextAddr    uint64
}

type Image struct {
	Name        string
	Description string
	Type        string
	Arch        string
	Os          string
	Compression string
	LoadAddr    uint64
	Data        []byte
}

func (f *Fit) getProperty(n *fdt.Node, propName string) []byte {
	if val, ok := n.Properties[propName]; ok {
		return val
	}

	panic(fmt.Errorf("Required property %s missing\n", propName))
}

// validateHash takes a hash node, and attempts to validate it. It takes
func (f *Fit) validateHash(n *fdt.Node, i *Image) (err error) {
	algo := f.getProperty(n, "algo")
	value := f.getProperty(n, "value")
	algostr := f.fdt.PropString(algo)

	if f.Debug {
		fmt.Printf("Checking %s:%s %v... ", i.Name, algostr, value)
	}
	if algostr == "sha1" {
		shasum := sha1.Sum(i.Data)
		shaslice := shasum[:]
		if !bytes.Equal(value, shaslice) {
			if f.Debug {
				fmt.Printf("error, calculated %v!\n", shaslice)
			}
			return fmt.Errorf("sha1 incorrect, expected %v! calculated %v!\n", value, shaslice)
		}
		if f.Debug {
			fmt.Print("OK!\n")
		}
		return
	}

	if algostr == "crc32" {
		propsum := f.fdt.PropUint32(value)
		calcsum := crc32.ChecksumIEEE(i.Data)
		if calcsum != propsum {
			if f.Debug {
				fmt.Printf("incorrect, expected %d calculated %d", propsum, calcsum)
			}
			return fmt.Errorf("crc32 incorrect, expected %d calculated %d", propsum, calcsum)
		}
		if f.Debug {
			fmt.Printf("OK!\n")
		}
		return
	}

	if algostr == "md5" {
		md5sum := md5.Sum(i.Data)
		md5slice := md5sum[:]
		if !bytes.Equal(value, md5slice) {
			if f.Debug {
				fmt.Printf("error, calculated %v!\n", md5slice)
			}
			return fmt.Errorf("sha1 incorrect, expected %v! calculated %v!\n", value, md5slice)
		}
		if f.Debug {
			fmt.Print("OK!\n")
		}
		return
	}

	if f.Debug {
		fmt.Printf("Unknown algorithm!\n")
	}
	return
}

func (f *Fit) validateHashes(n *fdt.Node, i *Image) (err error) {
	for _, c := range n.Children {
		if c.Name == "hash" || strings.HasPrefix(c.Name, "hash@") {
			err = f.validateHash(c, i)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Fit) parseImage(cfg *Config, imageName string) {
	i := f.Images[imageName]

	cfg.ImageList = append(cfg.ImageList, i)
}

func (f *Fit) parseConfiguration(whichconf string) (err error) {
	cfg := Config{}

	conf := f.fdt.RootNode.Children["configurations"]

	if f.Debug {
		fmt.Printf("parseConfiguration %s: %q\n", conf.Name, whichconf)
	}

	conf, ok := conf.Children[whichconf]

	if !ok {
		return fmt.Errorf("Can't find configuration %s", whichconf)
	}

	description := conf.Properties["description"]
	if description != nil {
		if f.Debug {
			fmt.Printf("parseConfiguration %s: %s\n", whichconf, f.fdt.PropString(description))
		}
	}

	cfg.ImageList = []*Image{}

	kernel := f.fdt.PropString(conf.Properties["kernel"])
	fdt := f.fdt.PropString(conf.Properties["fdt"])
	ramdisk := f.fdt.PropString(conf.Properties["ramdisk"])

	if f.Debug {
		fmt.Printf("parseConfiguration kernel=%s fdt=%s ramdisk=%s\n", kernel, fdt, ramdisk)
	}
	f.parseImage(&cfg, kernel)
	f.parseImage(&cfg, fdt)
	f.parseImage(&cfg, ramdisk)

	f.Configs[whichconf] = &cfg

	return nil
}

func Parse(b []byte) (f *Fit) {
	fit := Fit{}
	f = &fit
	f.fdt = &fdt.Tree{Debug: false, IsLittleEndian: false}
	err := f.fdt.Parse(b)
	if err != nil {
		panic(err)
	}

	f.Description = f.fdt.PropString(f.getProperty(f.fdt.RootNode, "description"))
	f.AddressCells = f.fdt.PropUint32(f.getProperty(f.fdt.RootNode, "#address-cells"))
	f.TimeStamp = time.Unix(int64(f.fdt.PropUint32(f.getProperty(f.fdt.RootNode, "timestamp"))), 0)

	images := f.fdt.RootNode.Children["images"]
	f.Images = make(map[string]*Image)

	for _, image := range images.Children {
		i := Image{}
		i.Name = image.Name
		i.Description = f.fdt.PropString(f.getProperty(image, "description"))
		i.Type = f.fdt.PropString(f.getProperty(image, "type"))
		i.Arch = f.fdt.PropString(f.getProperty(image, "arch"))
		i.Os = f.fdt.PropString(image.Properties["os"])
		i.Compression = f.fdt.PropString(f.getProperty(image, "compression"))
		i.Data = f.getProperty(image, "data")

		err := f.validateHashes(image, &i)
		if err != nil {
			panic(err)
		}
		load := f.fdt.PropUint32Slice(image.Properties["load"])
		entry := f.fdt.PropUint32Slice(image.Properties["entry"])

		if len(load) != 0 {
			if f.Debug {
				fmt.Printf("image %s: load=%x entry=%x len=%x\n", image.Name, load[0], entry[0], len(i.Data))
			}
			i.LoadAddr = uint64(load[0]) // fixme #address-cells
		}
		f.Images[image.Name] = &i
	}

	conf := f.fdt.RootNode.Children["configurations"]
	f.Configs = make(map[string]*Config)

	f.DefaultConfig = f.fdt.PropString(f.getProperty(conf, "default"))

	for _, c := range conf.Children {
		if c.Name == "conf" || strings.HasPrefix(c.Name, "conf@") {
			err := f.parseConfiguration(c.Name)
			if err != nil {
				panic(err)
			}
		}
	}

	return
}
