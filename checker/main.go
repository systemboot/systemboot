package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/systemboot/systemboot/pkg/checker"
)

var (
	configFile = flag.String("config-file", "", "Config file to use")
)

func main() {
	flag.Parse()
	// if flag.NArg() != 2 {
	//   flag.Usage()
	//   os.Exit(1)
	// }

	log.Printf("Registered: %v\n", checker.ListRegistered())

	var checklist []checker.Check

	checkerConfigStr, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Printf("Unable to open config file %#v: %s", *configFile, err.Error())
		os.Exit(1)
	}

	err = json.Unmarshal(checkerConfigStr, &checklist)
	if err != nil {
		log.Printf("Unable to parse config file %#v: %s\n", *configFile, err.Error())
		os.Exit(1)
	}

	checklistJSON, _ := json.MarshalIndent(checklist, "", "    ")
	log.Printf("Checklist: %s\n", checklistJSON)

	results, numErrors := checker.Run(checklist)
	resultsJSON, _ := json.MarshalIndent(results, "", "    ")
	fmt.Printf("Checker Results: %s\n", resultsJSON)

	if numErrors > 0 {
		os.Exit(1)
	}
}
