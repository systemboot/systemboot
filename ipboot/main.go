package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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

type netVars struct {
	HostIP         string `json:"host_ip"`
	HostNetmask    string `json:"netmask"`
	DefaultGateway string `json:"gateway"`
	DNSServer      string `json:"dns"`

	HostPrivKey string `json:"host_priv_key"`
	HostPupKey  string `json:"host_pub_key"`

	BootstrapURL    string `json:"bootstrap_url"`
	SignaturePubKey string `json:"signature_pub_key"`
}

func main() {
	flag.Parse()
	if *doDebug {
		debug = log.Printf
	}
	log.Print(banner)

	//get network variables
	file, err := os.Open("/root/netvars.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	data, _ := ioutil.ReadAll(file)
	fmt.Println(string(data))
	vars := netVars{}
	json.Unmarshal(data, &vars)
	fmt.Printf("Parsed network variables: %v\n", vars)

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
