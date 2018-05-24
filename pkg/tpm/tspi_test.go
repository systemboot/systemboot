package tpm

import (
	"testing"
	//"github.com/stretchr/testify/require"
)

func TestReadPcrTPM1(t *testing.T) {
	SetupTPM()
	PrintInfo()
}
