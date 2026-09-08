package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/volcengine/byted-postgresql-cli/internal/volcengine"
)

func TestValidateComputeType(t *testing.T) {
	for _, role := range []string{volcengine.ComputeRoleReadOnly, volcengine.ComputeRoleAnalytic} {
		if err := validateComputeType(role); err != nil {
			t.Fatalf("validateComputeType(%q) error = %v", role, err)
		}
	}
	normalizedRole := strings.TrimSpace(" ReadOnly ")
	if err := validateComputeType(normalizedRole); err != nil {
		t.Fatalf("validateComputeType(%q) error = %v", normalizedRole, err)
	}
	if normalizedRole != volcengine.ComputeRoleReadOnly {
		t.Fatalf("normalized role = %q, want %q", normalizedRole, volcengine.ComputeRoleReadOnly)
	}
	for _, invalid := range []string{"", volcengine.ComputeRolePrimary} {
		if err := validateComputeType(invalid); err == nil ||
			!strings.Contains(err.Error(), "--type must be") {
			t.Fatalf("validateComputeType(%q) error = %v", invalid, err)
		}
	}
}

func TestValidateComputeCreateUnits(t *testing.T) {
	for _, test := range []struct {
		name      string
		minCU     float64
		maxCU     float64
		setMin    bool
		setMax    bool
		wantError string
	}{
		{name: "valid lower bound", minCU: 0.25, maxCU: 0.25, setMin: true, setMax: true},
		{name: "valid upper bound", minCU: 0.25, maxCU: 2, setMin: true, setMax: true},
		{name: "min too low", minCU: 0.1, maxCU: 1, setMin: true, setMax: true, wantError: "--min-cu"},
		{name: "max too high", minCU: 0.25, maxCU: 2.1, setMin: true, setMax: true, wantError: "--max-cu"},
		{name: "max below min", minCU: 1, maxCU: 0.25, setMin: true, setMax: true, wantError: "greater than or equal"},
		{name: "min missing", minCU: 0, maxCU: 1, setMax: true, wantError: "required"},
		{name: "max missing", minCU: 0.25, maxCU: 0, setMin: true, wantError: "required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var command *cobra.Command
			for _, candidate := range newComputesCmd(defaultProviderContext()).Commands() {
				if candidate.Name() == "create" {
					command = candidate
					break
				}
			}
			if command == nil {
				t.Fatal("computes create command not found")
			}
			if test.setMin {
				if err := command.Flags().Set("min-cu", fmt.Sprintf("%g", test.minCU)); err != nil {
					t.Fatal(err)
				}
			}
			if test.setMax {
				if err := command.Flags().Set("max-cu", fmt.Sprintf("%g", test.maxCU)); err != nil {
					t.Fatal(err)
				}
			}
			err := validateComputeCreateUnits(command, test.minCU, test.maxCU)
			if test.wantError == "" && err != nil {
				t.Fatalf("validateComputeCreateUnits() error = %v", err)
			}
			if test.wantError != "" && (err == nil || !strings.Contains(err.Error(), test.wantError)) {
				t.Fatalf("validateComputeCreateUnits() error = %v, want %q", err, test.wantError)
			}
		})
	}
}
