package main

// https://xkcd.com/927/

import (
	"log"
)

const (
	// Author is the author
	Author = "Philipp Deppenwiese"
	// HelpText is the command line help
	HelpText = "A tool for managing boot configurations"
)

var goversion string

var (
	genkeys = kingpin.Command("genkeys", "Generate RSA keypair")
	pack    = kingpin.Command("pack", "Create boot configuration file")
	unpack  = kingpin.Command("unpack", "Unpack boot configuration file into directory")

	genkeysPrivateKeyFile = genkeys.Arg("privateKey", "File path to write the private key").Required().String()
	genkeysPublicKeyFile  = genkeys.Arg("publicKey", "File path to write the public key").Required().String()
	genkeysPassphrase     = genkeys.Flag("passphrase", "Encrypt keypair in PKCS8 format").String()

	packSignPassphrase     = pack.Flag("passphrase", "Passphrase for private key file").String()
	packKernelsDir         = pack.Flag("kernel-dir", "Path to the kernel directory containing kernel files").String()
	packInitrdsDir         = pack.Flag("initrd-dir", "Path to the initrd directory containing initrd files").String()
	packDTsDir             = pack.Flag("dt-dir", "Path to the dt directory containing device tree files").String()
	packManifest           = pack.Arg("manifest", "Path to the manifest file in JSON format").Required().String()
	packOutputFilename     = pack.Arg("bc-file", "Path to output file").Required().String()
	packSignPrivateKeyFile = pack.Arg("private-key", "Path to the private key file").String()

	unpackInputFilename       = unpack.Arg("bc-file", "Boot configuration file").Required().String()
	unpackDir                 = unpack.Arg("output-dir", "Path to the unpacked output directory").Required().String()
	unpackVerifyPublicKeyFile = unpack.Arg("public-key", "Path to the public key file").String()
)

func main() {
	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version(goversion).Author(Author)
	kingpin.CommandLine.Help = HelpText

	switch kingpin.Parse() {
	case "genkeys":
		if err := GenKeys(); err != nil {
			log.Fatalln(err.Error())
		}
	case "pack":
		if err := PackBootConfiguration(); err != nil {
			log.Panicln(err.Error())
		}
	case "unpack":
		if err := UnpackBootConfiguration(); err != nil {
			log.Fatalln(err.Error())
		}
	default:
		log.Fatal("Command not found")
	}
}
