package rng

import (
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

// The concept
//
// Most systems in real world application do not provide enough entropy at boot
// time. Therefore we will seed /dev/random with /dev/hwrng if a HW random
// number generator is available. Entropy is important for cryptographic
// protocols running in network stacks. Also disk encryption can be a problem
// if bad or no entropy is available. It can either block provisioning or makes
// a symmetric key easy to re-calculate.

var (
	// HwRandomCurrentFile shows/sets the current
	// HW random number generator
	HwRandomCurrentFile = "/sys/class/misc/hw_random/rng_current"
	// HwRandomAvailableFile shows the current available
	// HW random number generator
	HwRandomAvailableFile = "/sys/class/misc/hw_random/rng_available"
	// RandomPoolSizeFile returns the entropy poolsize
	RandomPoolSizeFile = "/proc/sys/kernel/random/poolsize"
	// RandomEntropyAvailableFile shows how much of the entropy poolsize is used
	RandomEntropyAvailableFile = "/proc/sys/kernel/random/entropy_avail"
	// EntropyFeedTime sets the loop time for seeding /dev/random by /dev/hwrng
	// in seconds
	EntropyFeedTime time.Duration = 10
	// RandomDevice is the linux random device
	RandomDevice = "/dev/random"
	// HwRandomDevice is the linux hw random device
	HwRandomDevice = "/dev/hwrng"
)

// Can be extended but keep in mind to priorize
// more secure random sources like hw random over
// timer, jitter based mechanisms. Zero is the highest
// priority.
// <rng-name> : <priority>
var trngList = map[string]int{
	"tpm-rng":        0,
	"intel-rng":      1,
	"amd-rng":        1,
	"timeriomem-rng": 2,
}

// Searches for available True Random Number Generator
// inside the kernel api and sets the most secure on if
// available which seeds /dev/hwrng
func setAvailableTRNG() (bool, error) {
	var (
		currentRNG    string
		availableRNGs []string
		rngs          []string
	)

	currentFileData, err := ioutil.ReadFile(HwRandomCurrentFile)
	if err != nil {
		return false, err
	}
	currentRNG = string(currentFileData)

	availableFileData, err := ioutil.ReadFile(HwRandomAvailableFile)
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

	err = ioutil.WriteFile(HwRandomCurrentFile, []byte(rngs[0]), 0644)
	if err != nil {
		return false, err
	}

	return true, nil
}

// UpdateLinuxRandomness seeds random data from
// /dev/hwrng into /dev/random based on a timer and
// the entropy pool size
// Usage:
// go UpdateLinuxRandomness()
func UpdateLinuxRandomness() error {
	good, err := setAvailableTRNG()
	if !good {
		return err
	}

	randomPoolSizeData, err := ioutil.ReadFile(RandomPoolSizeFile)
	if err != nil {
		return err
	}

	formatted := strings.TrimSuffix(string(randomPoolSizeData), "\n")
	randomPoolSize, err := strconv.ParseUint(formatted, 10, 32)
	if err != nil {
		return err
	}

	hwRng, err := os.OpenFile(HwRandomDevice, os.O_RDONLY, os.ModeDevice)
	if err != nil {
		return err
	}

	defer hwRng.Close()

	rng, err := os.OpenFile(RandomDevice, os.O_APPEND|os.O_WRONLY, os.ModeDevice)
	if err != nil {
		return err
	}

	defer rng.Close()

	for {
		randomEntropyAvailableData, err := ioutil.ReadFile(RandomEntropyAvailableFile)
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

		time.Sleep(EntropyFeedTime * time.Second)
	}
}
