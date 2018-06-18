package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFindMountPointNoError checks that there is no
// error in parsong the test output.
func TestFindMountPointNoError(t *testing.T) {
	LinuxMountsPath = "tests/mounts"
	_, err := GetMountpointByDevice("/dev/mapper/sys-old")
	require.NoError(t, err)
}

// TestFindMountPointValid check for valid output of
// test mountpoint.
func TestFindMountPointValid(t *testing.T) {
	LinuxMountsPath = "tests/mounts"
	mountpoint, err := GetMountpointByDevice("/dev/mapper/sys-old")
	require.NoError(t, err)
	require.Equal(t, mountpoint, "/media/usb")
}
