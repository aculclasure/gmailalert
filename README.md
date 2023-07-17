# gmailalert

gmailalert is a Go package and command-line app that scans a Gmail user's account for emails matching specified patterns and emits Pushover alerts when matches are found.

## Installation
1. Install the gmailalert command-line app by running `go install https://github.com/elskohder/gmailalert/cmd/gmailalert@latest`
2. You will need to create a [Google Cloud Project with the API enabled](https://developers.google.com/workspace/guides/create-project).
3. You will need to [create access credentials](https://developers.google.com/workspace/guides/create-credentials) for a Google desktop application and download them to your local machine. These are typically saved in a file called `credentials.json`.
4. You will need to have a [Pushover](https://pushover.net/) account with at least one application configured in it. See the [Pushover support page](https://support.pushover.net/) for more help setting up a Pushover application.

## Usage
```
$ ./gmailalert -h
Usage of gmailalert:
  -alerts-cfg-file string
        json file containing the alerting criteria (default "alerts.json")
  -credentials-file string
        json file containing your Google Developers Console credentials (default "credentials.json")
  -debug
        enable debug-level-logging
  -port int
        the port for the local http server to listen on for redirects from the Gmail OAuth2 resource provider (default 9999)
  -save-token
        save remotely fetched oauth2 token into the file specified in the -token-file flag
  -token-file string
        json file to read your Gmail OAuth2 token from (if present), or to save your Gmail OAuth2 token into (if not present) (default "token.json")
```

The gmailalert app reads a JSON configuration file containing email matching criteria (in [Gmail query format](https://support.google.com/mail/answer/7190?hl=en)) and the corresponding Pushover message to send when matches occur. This JSON configuration file is specified with the `-alerts-cfg` flag in the gmailalert command-line app.

Here is an example configuration:
```
{
    "pushoverapp": "NOT SHOWN HERE",
    "alerts": [
        {   
            "gmailquery": "is:unread subject:\"Your Bill is Available Online\"",     
            "pushovertarget": "NOT SHOWN HERE",
            "pushovertitle": "Bill Due!",
            "pushoversound": "cashregister"
        },
        {   
            "gmailquery": "is:unread subject:\"Your zoom meeting has started\"",     
            "pushovertarget": "NOT SHOWN HERE",
            "pushovertitle": "Zoom Meeting Started!",
            "pushoversound": "siren"
        }
    ]
}
```
Some points to note here:
- The value of the "pushoverapp" field is the API token for the pushover application that you want to emit notifications with.
- The value of the "pushovertarget" field is your Pushover account user key.

For example, assuming the JSON configuration shown above is saved in a file called `alerts.json`:

```
$ ./gmailalert -alerts-cfg-file alerts.json 
Processing 2 email queries to determine if any alerts will be emitted...
Emitted 0 alerts
```

## References
- [quickstart code from Google](https://github.com/googleworkspace/go-samples/blob/main/gmail/quickstart/quickstart.go)
- [quickstart article from Google](https://developers.google.com/gmail/api/quickstart/go)
