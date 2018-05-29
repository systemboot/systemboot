package rng

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	trngList map[string]int
)

const (
	hwRandomDir                string        = "/sys/class/misc/hw_random"
	hwRandomCurrentFile        string        = "/sys/class/misc/hw_random/rng_current"
	hwRandomAvailableFile      string        = "/sys/class/misc/hw_random/rng_available"
	randomPoolSizeFile         string        = "/proc/sys/kernel/random/poolsize"
	randomEntropyAvailableFile string        = "/proc/sys/kernel/random/entropy_avail"
	entropyFeedTime            time.Duration = 10
	randomDevice               string        = "/dev/random"
	hwRandomDevice             string        = "/dev/hwrng"
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
	currentRNG = string(currentFileData)

	availableFileData, err := ioutil.ReadFile(hwRandomAvailableFile)
	if err != nil {
		return false, err
	}
	availableRNGs = strings.Split(string(availableFileData), " ")

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
		return err
	}

	randomPoolSizeData, err := ioutil.ReadFile(randomPoolSizeFile)
	if err != nil {
		return err
	}

	formatted := strings.TrimSuffix(string(randomPoolSizeData), "\n")
	randomPoolSize, err := strconv.ParseUint(formatted, 10, 32)
	if err != nil {
		return err
	}

	hwRng, err := os.OpenFile(hwRandomDevice, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return err
	}

	defer hwRng.Close()

	rng, err := os.OpenFile(randomDevice, os.O_APPEND|os.O_WRONLY, os.ModeDevice)
	if err != nil {
		return err
	}

	defer rng.Close()

	for {
		randomEntropyAvailableData, err := ioutil.ReadFile(randomEntropyAvailableFile)
		if err != nil {
			return err
		}

		formatted := strings.TrimSuffix(string(randomEntropyAvailableData), "\n")
		randomEntropyAvailable, err := strconv.ParseUint(formatted, 10, 32)
		if err != nil {
			return err
		}

		randomBytesNeeded := randomPoolSize - randomEntropyAvailable
		if randomBytesNeeded > 0 {
			var random = make([]byte, randomBytesNeeded)
			if _, err = hwRng.Read(random); err != nil {
				return err
			}
			if _, err = rng.Write(random); err != nil {
				return err
			}
		}

		time.Sleep(entropyFeedTime * time.Second)
	}
}
