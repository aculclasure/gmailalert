package gmailalert_test

import (
	"testing"

	"github.com/aculclasure/gmailalert"
)

func TestCLIWithInvalidArgsReturnsError(t *testing.T) {
	t.Parallel()

	commandLineArgs := []string{"-alerts-cfg-file=", "-credentials-file="}

	if err := gmailalert.CLI(commandLineArgs); err == nil {
		t.Error("expected an error but did not get one")
	}
}
