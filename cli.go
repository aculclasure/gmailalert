package gmailalert

import (
	"errors"
	"flag"
	"io"
	"log"
	"os"
)

// CLI accepts a slice of command-line flags for a user's Google Developers
// Console file ("-credentials-file"), a user's local Google OAuth2 token JSON file
// ("-token-file"), an alert configuration JSON file ("-alerts-cfg-file") which
// provides the email criteria to alert on, a TCP port for the local HTTP server
// to listen on for redirect requests from the Google OAuth2 resource provider
// ("-port"), and a debug flag ("-debug") which indicates if debug-level output
// will be written.
//
// The command line flags are parsed, validated, and then used to create an
// Alerter struct to process alerts with. An error is returned if any of the
// command-line flags are invalid or if there is a problem during the processing
// of alerts.
func CLI(args []string) error {
	var app cliEnv

	if err := app.fromArgs(args); err != nil {
		return err
	}

	f, err := os.Open(app.alertsConfigFile)
	if err != nil {
		return err
	}
	defer f.Close()
	alertCfg, err := DecodeAlerts(f)
	if err != nil {
		return err
	}

	debugLogger := log.New(io.Discard, "", log.LstdFlags)
	if app.debug {
		debugLogger = log.New(os.Stdout, "DEBUG: ", log.LstdFlags|log.Lshortfile)
	}

	gmailClient, err := NewGmailClient(
		GmailClientConfig{
			CredentialsFile: app.credsFile,
			TokenFile:       app.tokenFile,
			UserInput:       os.Stdin,
			RedirectSvrPort: app.redirectSvrPort,
			Logger:          debugLogger,
		},
	)
	if err != nil {
		return err
	}

	pushoverClient, err := NewPushoverClient(alertCfg.PushoverApp, WithPushoverClientLogger(debugLogger))
	if err != nil {
		return err
	}

	alerter, err := NewAlerter(gmailClient, pushoverClient)
	if err != nil {
		return err
	}

	if err := alerter.Process(alertCfg.Alerts); err != nil {
		return err
	}

	return nil
}

// cliEnv is a type representing the CLI application environment.
type cliEnv struct {
	alertsConfigFile string
	credsFile        string
	tokenFile        string
	redirectSvrPort  int
	debug            bool
}

// fromArgs accepts a slice of command line flags, parses them, and encodes
// them into the given appEnv receiver. An error is returned if a problem
// is encountered during parsing or if any of the given command line flags
// has an empty value.
func (c *cliEnv) fromArgs(args []string) error {
	fs := flag.NewFlagSet("gmailalert", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(
		&c.alertsConfigFile,
		"alerts-cfg-file",
		"alerts.json",
		"json file containing the alerting criteria")
	fs.StringVar(
		&c.credsFile,
		"credentials-file",
		"credentials.json",
		"json file containing your Google Developers Console credentials")
	fs.StringVar(
		&c.tokenFile,
		"token-file",
		"token.json",
		"json file to read your Gmail OAuth2 token from (if present), or to save your Gmail OAuth2 token into (if not present)")
	fs.IntVar(
		&c.redirectSvrPort,
		"port",
		9999,
		"the port for the local http server to listen on for redirects from the Gmail OAuth2 resource provider",
	)
	fs.BoolVar(
		&c.debug,
		"debug",
		false,
		"enable debug-level-logging")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if c.credsFile == "" || c.alertsConfigFile == "" {
		fs.Usage()
		return errors.New(`command line flags "-credentials-file" "-alerts-cfg-file" must be non-empty`)
	}

	return nil
}
