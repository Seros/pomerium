package proxy

import (
	"crypto/cipher"
	"net/url"

	"github.com/pomerium/pomerium/config"
	"github.com/pomerium/pomerium/internal/encoding"
	"github.com/pomerium/pomerium/internal/encoding/jws"
	"github.com/pomerium/pomerium/internal/sessions"
	"github.com/pomerium/pomerium/internal/sessions/cookie"
	"github.com/pomerium/pomerium/pkg/cryptutil"
)

type proxyState struct {
	sharedKey    []byte
	sharedCipher cipher.AEAD

	authenticateURL          *url.URL
	authenticateDashboardURL *url.URL
	authenticateSigninURL    *url.URL
	authenticateRefreshURL   *url.URL

	encoder         encoding.MarshalUnmarshaler
	cookieSecret    []byte
	sessionStore    sessions.SessionStore
	jwtClaimHeaders config.JWTClaimHeaders

	programmaticRedirectDomainWhitelist []string
}

func newProxyStateFromConfig(cfg *config.Config) (*proxyState, error) {
	err := ValidateOptions(cfg.Options)
	if err != nil {
		return nil, err
	}

	state := new(proxyState)
	state.sharedKey, err = cfg.Options.GetSharedKey()
	if err != nil {
		return nil, err
	}

	state.sharedCipher, err = cryptutil.NewAEADCipher(state.sharedKey)
	if err != nil {
		return nil, err
	}

	state.cookieSecret, err = cfg.Options.GetCookieSecret()
	if err != nil {
		return nil, err
	}

	// used to load and verify JWT tokens signed by the authenticate service
	state.encoder, err = jws.NewHS256Signer(state.sharedKey)
	if err != nil {
		return nil, err
	}

	state.jwtClaimHeaders = cfg.Options.JWTClaimsHeaders

	// errors checked in ValidateOptions
	state.authenticateURL, err = cfg.Options.GetAuthenticateURL()
	if err != nil {
		return nil, err
	}

	state.authenticateDashboardURL = state.authenticateURL.ResolveReference(&url.URL{Path: "/.pomerium/"})
	state.authenticateSigninURL = state.authenticateURL.ResolveReference(&url.URL{Path: signinURL})
	state.authenticateRefreshURL = state.authenticateURL.ResolveReference(&url.URL{Path: refreshURL})

	state.sessionStore, err = cookie.NewStore(func() cookie.Options {
		return cookie.Options{
			Name:     cfg.Options.CookieName,
			Domain:   cfg.Options.CookieDomain,
			Secure:   cfg.Options.CookieSecure,
			HTTPOnly: cfg.Options.CookieHTTPOnly,
			Expire:   cfg.Options.CookieExpire,
		}
	}, state.encoder)
	if err != nil {
		return nil, err
	}
	state.programmaticRedirectDomainWhitelist = cfg.Options.ProgrammaticRedirectDomainWhitelist

	return state, nil
}
