package application

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"flamingo.me/flamingo/framework/flamingo"
	"flamingo.me/flamingo/framework/router"
	"flamingo.me/flamingo/framework/web"

	"flamingo.me/flamingo/core/auth/domain"
	"github.com/coreos/go-oidc"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	// KeyToken defines where the authentication token is saved
	KeyToken = "auth.token"

	// KeyRawIDToken defines where the raw ID token is saved
	KeyRawIDToken = "auth.rawidtoken"

	// KeyAuthstate defines the current internal authentication state
	KeyAuthstate = "auth.state"
)

type (
	// AuthManager handles authentication related operations
	AuthManager struct {
		Server              string          `inject:"config:auth.server"`
		Secret              string          `inject:"config:auth.secret"`
		ClientID            string          `inject:"config:auth.clientid"`
		MyHost              string          `inject:"config:auth.myhost"`
		DisableOfflineToken bool            `inject:"config:auth.disableOfflineToken"`
		Logger              flamingo.Logger `inject:""`

		NotBefore time.Time

		Router *router.Router `inject:""`

		openIDProvider *oidc.Provider
		oauth2Config   *oauth2.Config
	}
)

// Auth tries to retrieve the authentication context for a active session
func (am *AuthManager) Auth(c web.Context) (domain.Auth, error) {
	ts, err := am.TokenSource(c)
	if err != nil {
		return domain.Auth{}, err
	}
	idToken, err := am.IDToken(c)
	if err != nil {
		return domain.Auth{}, err
	}

	return domain.Auth{
		TokenSource: ts,
		IDToken:     idToken,
	}, nil
}

// OpenIDProvider is a lazy initialized OID provider
func (am *AuthManager) OpenIDProvider() *oidc.Provider {
	if am.openIDProvider == nil {
		var err error
		am.openIDProvider, err = oidc.NewProvider(context.Background(), am.Server)
		if err != nil {
			panic(err)
		}
	}
	return am.openIDProvider
}

// OAuth2Config is lazy setup oauth2config
func (am *AuthManager) OAuth2Config() *oauth2.Config {
	if am.oauth2Config != nil {
		return am.oauth2Config
	}

	callbackURL := am.Router.URL("auth.callback", nil)

	am.Logger.WithField(flamingo.LogKeyCategory, "auth").Debug("am Callback", am, callbackURL)

	myhost, err := url.Parse(am.MyHost)
	if err != nil {
		am.Logger.WithField(flamingo.LogKeyCategory, "auth").Error("Url parse failed", am.MyHost, err)
	}
	callbackURL.Host = myhost.Host
	callbackURL.Scheme = myhost.Scheme
	scopes := []string{oidc.ScopeOpenID, "profile", "email"}
	if !am.DisableOfflineToken {
		scopes = append(scopes, oidc.ScopeOfflineAccess)
	}

	am.oauth2Config = &oauth2.Config{
		ClientID:     am.ClientID,
		ClientSecret: am.Secret,
		RedirectURL:  callbackURL.String(),

		// Discovery returns the OAuth2 endpoints.
		// It might panic here if Endpoint cannot be discovered
		Endpoint: am.OpenIDProvider().Endpoint(),

		// "openid" is a required scope for OpenID Connect flows.
		Scopes: scopes,
	}
	am.Logger.WithField(flamingo.LogKeyCategory, "auth").Debug("am.oauth2Config", am.oauth2Config)
	return am.oauth2Config
}

// Verifier creates an OID verifier
func (am *AuthManager) Verifier() *oidc.IDTokenVerifier {
	return am.OpenIDProvider().Verifier(&oidc.Config{ClientID: am.ClientID})
}

// OAuth2Token retrieves the oauth2 token from the session
func (am *AuthManager) OAuth2Token(c web.Context) (*oauth2.Token, error) {
	if _, ok := c.Session().Values[KeyToken]; !ok {
		return nil, errors.New("no token")
	}

	oauth2Token, ok := c.Session().Values[KeyToken].(*oauth2.Token)
	if !ok {
		return nil, errors.Errorf("invalid token %T %v", c.Session().Values[KeyToken], c.Session().Values[KeyToken])
	}

	return oauth2Token, nil
}

// IDToken retrieves and validates the ID Token from the session
func (am *AuthManager) IDToken(c web.Context) (*oidc.IDToken, error) {
	token, _, err := am.getIDToken(c)
	return token, err
}

// GetRawIDToken gets the raw IDToken from session
func (am *AuthManager) GetRawIDToken(c web.Context) (string, error) {
	_, raw, err := am.getIDToken(c)
	return raw, err
}

// IDToken retrieves and validates the ID Token from the session
func (am *AuthManager) getIDToken(c web.Context) (*oidc.IDToken, string, error) {
	if c.Session() == nil {
		return nil, "", errors.New("no session configured")
	}

	if token, ok := c.Session().Values[KeyRawIDToken]; ok {
		idtoken, err := am.Verifier().Verify(c, token.(string))
		if err == nil {
			return idtoken, token.(string), nil
		}
	}

	token, raw, err := am.getNewIdToken(c)
	if err != nil {
		return nil, "", err
	}

	c.Session().Values[KeyRawIDToken] = raw

	return token, raw, nil
}

// IDToken retrieves and validates the ID Token from the session
func (am *AuthManager) getNewIdToken(c web.Context) (*oidc.IDToken, string, error) {
	tokenSource, err := am.TokenSource(c)
	if err != nil {
		return nil, "", errors.WithStack(err)
	}

	token, err := tokenSource.Token()
	if err != nil {
		return nil, "", errors.WithStack(err)
	}

	raw, err := am.ExtractRawIDToken(token)
	if err != nil {
		return nil, "", errors.WithStack(err)
	}

	idtoken, err := am.Verifier().Verify(c, raw)

	if idtoken == nil {
		return nil, "", errors.New("idtoken nil")
	}

	return idtoken, raw, err
}

// ExtractRawIDToken from the provided (fresh) oatuh2token
func (am *AuthManager) ExtractRawIDToken(oauth2Token *oauth2.Token) (string, error) {
	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return "", errors.Errorf("no id token %T %v", oauth2Token.Extra("id_token"), oauth2Token.Extra("id_token"))
	}

	return rawIDToken, nil
}

// TokenSource to be used in situations where you need it
func (am *AuthManager) TokenSource(c web.Context) (oauth2.TokenSource, error) {
	oauth2Token, err := am.OAuth2Token(c)
	if err != nil {
		return nil, err
	}

	return am.OAuth2Config().TokenSource(c, oauth2Token), nil
}

// HTTPClient to retrieve a client with automatic tokensource
func (am *AuthManager) HTTPClient(c web.Context) (*http.Client, error) {
	ts, err := am.TokenSource(c)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(c, ts), nil
}
