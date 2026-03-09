package cli

import (
	"errors"
	"fmt"
	"testing"
)

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "nil error",
			err:  nil,
			want: ExitSuccess,
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
			want: ExitUsageErr,
		},
		{
			name: "CLIError ExitValError",
			err:  NewCLIError(ExitValError, errors.New("validation failed")),
			want: ExitValError,
		},
		{
			name: "CLIError ExitConfigErr",
			err:  NewCLIError(ExitConfigErr, errors.New("config missing")),
			want: ExitConfigErr,
		},
		{
			name: "wrapped CLIError",
			err:  fmt.Errorf("wrapped: %w", NewCLIError(ExitConfigErr, errors.New("config missing"))),
			want: ExitConfigErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetExitCode(tt.err); got != tt.want {
				t.Errorf("GetExitCode() = %v, want %v", got, tt.want)
			}
		})
	}
}
