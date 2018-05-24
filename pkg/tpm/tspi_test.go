package tpm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadPcrTPM1(t *testing.T) {
	err := SetupTPM()
	require.NoError(t, err)
}
