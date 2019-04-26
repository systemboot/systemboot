package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
)

var (
	dryRun  = flag.Bool("dryrun", false, "Do everything except booting the loaded kernel")
	doDebug = flag.Bool("d", false, "Print debug output")
)

const (
	ip      = "10.0.2.15/24"
	gateway = "10.0.2.2/24"
	eth     = "eth0"
	url     = "http://mullvad.9esec.io/vmlinuz-fedora"
)

var banner = `
_____ _____    ____   ____   ____ _______ 
|_   _|  __ \  |  _ \ / __ \ / __ \__   __|
  | | | |__) | | |_) | |  | | |  | | | |   
  | | |  ___/  |  _ <| |  | | |  | | | |   
 _| |_| |      | |_) | |__| | |__| | | |   
|_____|_|      |____/ \____/ \____/  |_|   
`
var debug = func(string, ...interface{}) {}

func main() {
	flag.Parse()
	if *doDebug {
		debug = log.Printf
	}
	log.Print(banner)

	//setup ip
	cmd := exec.Command("ip", "addr", "add", ip, "dev", eth)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing %v: %v", cmd, err)
	}
	cmd = exec.Command("ip", "link", "set", eth, "up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing %v: %v", cmd, err)
	}
	cmd = exec.Command("ip", "route", "add", "default", "via", gateway, "dev", eth)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing %v: %v", cmd, err)
	}

	//get kernel
	cmd = exec.Command("wget", "-O", "ipkernel", url)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing %v: %v", cmd, err)
	}

	// TODOcreate boot config

	//boot
	cmd = exec.Command("kexec", "-l", "ipkernel", "-c", "console=ttyS0,115200")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing %v: %v", cmd, err)
	}

	if *dryRun {
		debug("Dry-run mode: will not boot")
		return
	}

	cmd = exec.Command("kexec", "-e")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error executing %v: %v", cmd, err)
	}
}
