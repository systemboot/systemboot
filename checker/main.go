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

var configFile string

func init(){
	flag.StringVar(&configFile, "config-file", "", "Config file to use")
}

func usage(){
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func run() int {
	if configFile == "" {
		fmt.Fprintf(os.Stderr, "Error: config-file argument is required\n")
		usage()
		return 1
	}

	log.Printf("Registered Checks: %v\n", checker.ListRegistered())

	var checklist []checker.Check

	checkerConfigStr, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("Unable to open config file %#v: %s", configFile, err.Error())
		return 1
	}

	err = json.Unmarshal(checkerConfigStr, &checklist)
	if err != nil {
		log.Printf("Unable to parse config file %#v: %s\n", configFile, err.Error())
		return 1
	}

	checklistJSON, _ := json.MarshalIndent(checklist, "", "    ")
	log.Printf("Checklist: %s\n", checklistJSON)

	results, numErrors := checker.Run(checklist)
	resultsJSON, _ := json.MarshalIndent(results, "", "    ")
	fmt.Printf("Checker Results: %s\n", resultsJSON)

	if numErrors > 0 {
		return 1
	}

	return 0
}

func main() {
	flag.Parse()
	ret := run()
	os.Exit(ret)
}
