package main

import (
	"github.com/bizjs/kratoscarf/examples/app/auth-app/internal/conf"

	authjwt "github.com/bizjs/kratoscarf/auth/jwt"
	"github.com/bizjs/kratoscarf/auth/session"
)

func provideServerConfig(bc *conf.Bootstrap) *conf.ServerConfig {
	return &bc.Server
}

func provideJWTAuthenticator(bc *conf.Bootstrap) *authjwt.Authenticator {
	return authjwt.New(authjwt.Config{
		Secret:        bc.Auth.JWT.Secret,
		Issuer:        bc.Auth.JWT.Issuer,
		AccessExpiry:  bc.Auth.JWT.ParseAccessExpiry(),
		RefreshExpiry: bc.Auth.JWT.ParseRefreshExpiry(),
	})
}

func provideSessionManager(bc *conf.Bootstrap) *session.Manager {
	return session.NewManager(
		session.NewMemoryStore(),
		session.Config{
			MaxAge:     bc.Auth.Session.ParseMaxAge(),
			CookieName: bc.Auth.Session.CookieName,
		},
	)
}
