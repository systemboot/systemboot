package main

import (
	"flag"
	"log"
	"os/exec"
	"time"

	"github.com/insomniacslk/systemboot/pkg/booter"
)

var (
	doQuiet       = flag.Bool("q", false, "Disable verbose output")
	interval      = flag.Int("I", 1, "Interval in seconds before looping to the next boot command")
	noDefaultBoot = flag.Bool("nodefault", false, "Do not attempt default boot entries if regular ones fail")
)

var defaultBootsequence = [][]string{
	[]string{"netboot", "-userclass", "linuxboot"},
	[]string{"localboot"},
}

var supportedBooterParsers = []func([]byte) (booter.Booter, error){
	booter.NewNetBooter,
}

func main() {
	flag.Parse()

	log.Printf("*************************************************************************")
	log.Print("Starting boot sequence, press CTRL-C within 5 seconds to drop into a shell")
	log.Printf("*************************************************************************")
	time.Sleep(5 * time.Second)

	sleepInterval := time.Duration(*interval) * time.Second

	// Get and show boot entries
	bootEntries := booter.GetBootEntries()
	log.Printf("BOOT ENTRIES:")
	for _, entry := range bootEntries {
		log.Printf("    %v) %+v", entry.Name, string(entry.Config))
	}
	for _, entry := range bootEntries {
		log.Printf("Trying boot entry %s: %s", entry.Name, string(entry.Config))
		if err := entry.Booter.Boot(); err != nil {
			log.Printf("Warning: failed to boot with configuration: %+v", entry)
		}
		if !*doQuiet {
			log.Printf("Sleeping %v before attempting next boot command", sleepInterval)
		}
		time.Sleep(sleepInterval)
	}

	// if boot entries failed, use the default boot sequence
	log.Printf("Boot entries failed")

	if !*noDefaultBoot {
		log.Print("Falling back to the default boot sequence")
		for {
			for _, bootcmd := range defaultBootsequence {
				if !*doQuiet {
					bootcmd = append(bootcmd, "-d")
				}
				log.Printf("Running boot command: %v", bootcmd)
				cmd := exec.Command(bootcmd[0], bootcmd[1:]...)
				if err := cmd.Run(); err != nil {
					log.Printf("Error executing %v: %v", cmd, err)
				}
			}
			if !*doQuiet {
				log.Printf("Sleeping %v before attempting next boot command", sleepInterval)
			}
			time.Sleep(sleepInterval)
		}
	}
}
