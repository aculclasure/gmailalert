package gmail

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

type OAuth2 struct {
	GoogleCfg []byte
	cfg       *oauth2.Config
	tok       *oauth2.Token
}

func NewOAuth2(googleCfg io.Reader) (*OAuth2, error) {
	if googleCfg == nil {
		return nil, errors.New("google configuration must not be nil")
	}
	cfgBytes, err := io.ReadAll(googleCfg)
	if err != nil {
		return nil, fmt.Errorf("got unexpected error reading google configuration: %s", err)
	}

	if len(cfgBytes) == 0 {
		return nil, errors.New("google configuration must not be empty")
	}

	return &OAuth2{GoogleCfg: cfgBytes}, nil
}

func (o *OAuth2) LoadConfig() error {
	cfg, err := google.ConfigFromJSON(o.GoogleCfg, gmail.GmailReadonlyScope)
	if err != nil {
		return err
	}

	o.cfg = cfg
	return nil
}

func (o *OAuth2) LoadToken(token io.Reader) error {
	if token == nil {
		return errors.New("token must not be nil")
	}

	var tok oauth2.Token
	err := json.NewDecoder(token).Decode(&tok)
	if err != nil {
		return err
	}

	o.tok = &tok
	return nil
}

func (o *OAuth2) GetToken() ([]byte, error) {
	if o.tok == nil {
		return nil, errors.New("underlying oauth2 token must not be nil")
	}

	bfr := new(bytes.Buffer)
	err := json.NewEncoder(bfr).Encode(o.tok)
	if err != nil {
		return nil, err
	}

	return bfr.Bytes(), nil
}
