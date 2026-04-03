//go:build wireinject
// +build wireinject

package main

import (
	"github.com/bizjs/kratoscarf/examples/app/auth-app/internal/biz"
	"github.com/bizjs/kratoscarf/examples/app/auth-app/internal/conf"
	"github.com/bizjs/kratoscarf/examples/app/auth-app/internal/data"
	"github.com/bizjs/kratoscarf/examples/app/auth-app/internal/server"
	"github.com/bizjs/kratoscarf/examples/app/auth-app/internal/service"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

func wireApp(*conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(
		provideServerConfig,
		provideJWTAuthenticator,
		provideSessionManager,
		data.ProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		server.ProviderSet,
		newApp,
	))
}
