package webauth

import (
	"net/url"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/ipfans/authgate/utils/defaults"
)

type Config struct {
	DisplayName string `koanf:"display_name"`
	Origin      string `koanf:"origin"`
}

func Init(cfg Config) (*webauthn.WebAuthn, error) {
	u, err := url.Parse(cfg.Origin)
	if err != nil {
		return nil, err
	}
	config := &webauthn.Config{
		RPID:          defaults.Get(u.Host, "authgate"),
		RPDisplayName: defaults.Get(cfg.DisplayName, "AuthGate"),
		RPOrigins:     []string{defaults.Get(cfg.Origin, cfg.Origin)},
	}
	return webauthn.New(config)
}
