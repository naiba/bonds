package services

import (
	"encoding/json"
	"errors"
	"net/url"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/openidConnect"
	"golang.org/x/oauth2"
)

// Keep PKCE material on the individual Goth session, never on the shared
// provider: concurrent logins must not overwrite one another's verifier.
type pkceOIDCProvider struct{ *openidConnect.Provider }

type pkceOIDCSession struct {
	*openidConnect.Session
	Verifier string `json:"code_verifier"`
}

func (p *pkceOIDCProvider) BeginAuth(state string) (goth.Session, error) {
	session, err := p.Provider.BeginAuth(state)
	if err != nil {
		return nil, err
	}
	base := session.(*openidConnect.Session)
	authURL, err := url.Parse(base.AuthURL)
	if err != nil {
		return nil, err
	}
	verifier := oauth2.GenerateVerifier()
	params := authURL.Query()
	params.Set("code_challenge", oauth2.S256ChallengeFromVerifier(verifier))
	params.Set("code_challenge_method", "S256")
	authURL.RawQuery = params.Encode()
	base.AuthURL = authURL.String()
	return &pkceOIDCSession{Session: base, Verifier: verifier}, nil
}

func (p *pkceOIDCProvider) UnmarshalSession(data string) (goth.Session, error) {
	var session pkceOIDCSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, err
	}
	if session.Session == nil || session.Verifier == "" {
		return nil, errors.New("OIDC login session is missing PKCE material; restart sign-in")
	}
	return &session, nil
}

func (p *pkceOIDCProvider) FetchUser(session goth.Session) (goth.User, error) {
	return p.Provider.FetchUser(session.(*pkceOIDCSession).Session)
}

func (s *pkceOIDCSession) Marshal() string {
	// The embedded Goth session implements Marshal(), not json.Marshaler.
	data, _ := json.Marshal(s)
	return string(data)
}

func (s *pkceOIDCSession) Authorize(provider goth.Provider, params goth.Params) (string, error) {
	if s.Verifier == "" || params.Get("error") != "" {
		return "", errors.New("invalid OIDC authorization response")
	}
	// Never accept a callback-supplied verifier or redirect_uri override.
	safe := url.Values{"code": {params.Get("code")}, "code_verifier": {s.Verifier}}
	return s.Session.Authorize(provider.(*pkceOIDCProvider).Provider, safe)
}
