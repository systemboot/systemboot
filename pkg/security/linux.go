package security

import (
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var (
	trngList map[string]int
)

const (
	hwRandomDir           string        = "/sys/class/misc/hw_random"
	hwRandomCurrentFile   string        = "/sys/class/misc/hw_random/rng_current"
	hwRandomAvailableFile string        = "/sys/class/misc/hw_random/rng_available"
	entropyChunkSize      int64         = 128
	entropyFeedTime       time.Duration = 2
	randomDevice          string        = "/dev/random"
	hwRandomDevice        string        = "/dev/hwrng"
)

func init() {
	trngList = make(map[string]int)

	// Priorize the HW RNG 0 is the highest
	trngList["tpm-rng"] = 0
	trngList["intel-rng"] = 0
	trngList["amd-rng"] = 0
	trngList["timeriomem-rng"] = 3
}

func setAvailableTRNG() (bool, error) {
	var currentRNG string
	var availableRNGs []string
	var rngs []string
	if _, err := os.Stat(hwRandomDir); os.IsNotExist(err) {
		return false, err
	}

	currentFileData, err := ioutil.ReadFile(hwRandomCurrentFile)
	if err != nil {
		return false, err
	}
	currentRNG = string(currentFileData[:])

	availableFileData, err := ioutil.ReadFile(hwRandomAvailableFile)
	if err != nil {
		return false, err
	}
	availableRNGs = strings.Split(string(availableFileData[:]), " ")

	for key, value := range trngList {
		if key == currentRNG && value == 0 {
			return true, nil
		}
	}

	for _, rng := range availableRNGs {
		for key := range trngList {
			if rng == key {
				rngs = append(rngs, key)
			}
		}
	}

	if len(rngs) <= 0 {
		return false, nil
	}

	ioutil.WriteFile(hwRandomCurrentFile, []byte(rngs[0]), 0644)

	return true, nil
}

// UpdateLinuxRandomness should be executed
// as goroutine for constantly filling hw_random
// into the Linux random device. Function will auto
// block in case of no random in hw_rng.
func UpdateLinuxRandomness() error {
	good, err := setAvailableTRNG()
	if !good {
		Die("No valid TRNG config found. Details: "+err.Error(), false, 0)
	}

	hwRng, err := os.Open(hwRandomDevice)
	if err != nil {
		Die("Can't open hw random device. Details: "+err.Error(), false, 0)
	}

	rng, err := os.Open(randomDevice)
	if err != nil {
		Die("Can't open /dev/random device. Details: "+err.Error(), false, 0)
	}

	for {
		var random [entropyChunkSize]byte
		hwRng.Read(random[:])
		rng.Write(random[:])
		time.Sleep(entropyFeedTime * time.Second)
	}
}
