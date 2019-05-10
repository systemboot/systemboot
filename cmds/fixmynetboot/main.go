package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/systemboot/systemboot/pkg/checker"
)

// fixmynetboot is a troubleshooting tool that can help you identify issues that
// won't let your system boot over the network.
// NOTE: this is a DEMO tool. It's here only to show you how to write your own
// checks and remediations. Don't use it in production.

var emergencyShellBanner = `
**************************************************************************
** Interface checks failed, see the output above to debug the issue.     *
** Entering the emergency shell, where you can run "fixmynetboot" again, *
** or any other LinuxBoot command.                                       *
**************************************************************************
`

var (
	doEmergencyShell = flag.Bool("shell", false, "Run emergency shell if checks fail")
)

func checkInterface(ifname string) error {
	checklist := []checker.Check{
		checker.Check{
			Description:  fmt.Sprintf("%s exists", ifname),
			CheckFunName: "InterfaceExists",
			CheckFunArgs: checker.CheckArgs{"ifname": ifname},
			Remediations: []checker.Check{
				{
					CheckFunName: "InterfaceRemediate",
					CheckFunArgs: checker.CheckArgs{"ifname": ifname},
				},
			},
			StopOnFailure: true,
		},
		checker.Check{
			Description:   fmt.Sprintf("%s link speed", ifname),
			CheckFunName:  "LinkSpeed",
			CheckFunArgs:  checker.CheckArgs{"ifname": ifname, "minSpeed": "100"},
			StopOnFailure: false},
		checker.Check{
			Description:   fmt.Sprintf("%s link autoneg", ifname),
			CheckFunName:  "LinkAutoneg",
			CheckFunArgs:  checker.CheckArgs{"ifname": ifname, "expected": "true"},
			StopOnFailure: false,
		},
		checker.Check{
			Description:   fmt.Sprintf("%s has link-local", ifname),
			CheckFunName:  "InterfaceHasLinkLocalAddress",
			CheckFunArgs:  checker.CheckArgs{"ifname": ifname},
			StopOnFailure: true,
		},
		checker.Check{
			Description:   fmt.Sprintf("%s has global addresses", ifname),
			CheckFunName:  "InterfaceHasGlobalAddresses",
			CheckFunArgs:  checker.CheckArgs{"ifname": ifname},
			StopOnFailure: true,
		},
	}

	_, numErrors := checker.Run(checklist)

	if numErrors > 0 {
		return fmt.Errorf("%d checks failed", numErrors)
	}

	return nil
}

func getNonLoopbackInterfaces() ([]string, error) {
	var interfaces []string
	allInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range allInterfaces {
		if iface.Flags&net.FlagLoopback == 0 {
			interfaces = append(interfaces, iface.Name)
		}
	}
	return interfaces, nil
}

func main() {
	flag.Parse()
	var (
		interfaces []string
		err        error
	)
	ifname := flag.Arg(0)
	if ifname == "" {
		interfaces, err = getNonLoopbackInterfaces()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		interfaces = []string{ifname}
	}

	for _, ifname := range interfaces {
		if err := checkInterface(ifname); err != nil {
			if !*doEmergencyShell {
				log.Fatal(err)
			}
			if err := checker.EmergencyShell(checker.CheckArgs{"banner": emergencyShellBanner}); err != nil {
				log.Fatal(err)
			}
		}
	}
}
