package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func ThisCheckAlwaysSucceeds(args CheckArgs) error {
	return nil
}

func ThisCheckAlwaysFails(args CheckArgs) error {
	return fmt.Errorf("TEST FAILURE")
}

func init() {
	registerCheckFun(ThisCheckAlwaysSucceeds)
	registerCheckFun(ThisCheckAlwaysFails)
}

func TestRunSimpleOK(t *testing.T) {
	check := Check{
		Description:  "a_description",
		CheckFunName: "ThisCheckAlwaysSucceeds",
		CheckFunArgs: nil,
	}
	result := check.Run()
	require.Equal(t, check.Description, result.Description)
	require.Equal(t, check.CheckFunName, result.CheckFunName)
	require.Equal(t, check.CheckFunArgs, result.CheckFunArgs)
	require.Equal(t, result.Result, ResultOK)
	require.Equal(t, result.Error, "")
	require.Equal(t, len(result.RemediationResults), 0)
	require.Equal(t, result.StoppedOnFailure, false)
}

func TestRunSimpleError(t *testing.T) {
	check := Check{
		Description:  "a_description",
		CheckFunName: "ThisCheckAlwaysFails",
		CheckFunArgs: nil,
	}
	result := check.Run()
	require.Equal(t, check.Description, result.Description)
	require.Equal(t, check.CheckFunName, result.CheckFunName)
	require.Equal(t, check.CheckFunArgs, result.CheckFunArgs)
	require.Equal(t, ResultError, result.Result)
	require.Equal(t, result.Error, "TEST FAILURE")
	require.Equal(t, len(result.RemediationResults), 0)
	require.Equal(t, result.StoppedOnFailure, false)
}

func TestRunRemediation(t *testing.T) {
	check := Check{
		Description:  "a_description",
		CheckFunName: "ThisCheckAlwaysFails",
		CheckFunArgs: nil,
		Remediations: []Check{
			Check{
				CheckFunName: "ThisCheckAlwaysSucceeds",
			},
		},
	}
	result := check.Run()
	require.Equal(t, check.Description, result.Description)
	require.Equal(t, check.CheckFunName, result.CheckFunName)
	require.Equal(t, check.CheckFunArgs, result.CheckFunArgs)
	require.Equal(t, ResultError, result.Result)
	require.Equal(t, result.Error, "TEST FAILURE")
	require.Equal(t, len(result.RemediationResults), 1)
	require.Equal(t, result.RemediationResults[0].CheckFunName, "ThisCheckAlwaysSucceeds")
	require.Equal(t, result.RemediationResults[0].Error, "")
	require.Equal(t, result.RemediationResults[0].Result, ResultOK)
	require.Equal(t, result.StoppedOnFailure, false)
}

func TestRunStopOnFailure(t *testing.T) {
	check := Check{
		Description:   "a_description",
		CheckFunName:  "ThisCheckAlwaysFails",
		CheckFunArgs:  nil,
		StopOnFailure: true,
	}
	result := check.Run()
	require.Equal(t, len(result.RemediationResults), 0)
	require.Equal(t, true, result.StoppedOnFailure)
}

func TestRunStopOnFailureWithRemediations(t *testing.T) {
	check := Check{
		Description:  "a_description",
		CheckFunName: "ThisCheckAlwaysFails",
		CheckFunArgs: nil,
		Remediations: []Check{
			Check{
				CheckFunName: "ThisCheckAlwaysSucceeds",
			},
		},
		StopOnFailure: true,
	}
	result := check.Run()
	require.Equal(t, len(result.RemediationResults), 0)
	require.Equal(t, true, result.StoppedOnFailure)
}

func TestRunChecklist(t *testing.T) {
	checklist := []Check{
		Check{
			Description:  "a_description",
			CheckFunName: "ThisCheckAlwaysSucceeds",
			CheckFunArgs: nil,
		},
	}

	results, numErrors := Run(checklist)
	require.Equal(t, numErrors, 0)
	require.Equal(t, len(results), 1)
	require.Equal(t, checklist[0].Description, results[0].Description)
	require.Equal(t, checklist[0].CheckFunName, results[0].CheckFunName)
	require.Equal(t, checklist[0].CheckFunArgs, results[0].CheckFunArgs)
	require.Equal(t, results[0].Result, ResultOK)
}

func TestRunChecklistError(t *testing.T) {
	checklist := []Check{
		Check{
			Description:  "a_description",
			CheckFunName: "ThisCheckAlwaysFails",
			CheckFunArgs: nil,
		},
	}

	results, numErrors := Run(checklist)
	require.Equal(t, numErrors, 1)
	require.Equal(t, len(results), 1)
	require.Equal(t, checklist[0].Description, results[0].Description)
	require.Equal(t, checklist[0].CheckFunName, results[0].CheckFunName)
	require.Equal(t, checklist[0].CheckFunArgs, results[0].CheckFunArgs)
	require.Equal(t, results[0].Result, ResultError)
}
