package cli_test

import (
	"testing"

	"github.com/aculclasure/gmailalert/internal/ui/cli"
)

func TestRunWithInvalidArgsReturnsError(t *testing.T) {
	t.Parallel()

	commandLineArgs := []string{"-alerts-cfg-file=", "-credentials-file="}

	if err := cli.Run(commandLineArgs); err == nil {
		t.Error("expected an error but did not get one")
	}
}
