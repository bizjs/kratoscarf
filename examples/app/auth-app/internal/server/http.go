package server

import (
	v1 "github.com/bizjs/kratoscarf/examples/api/auth/v1"
	"github.com/bizjs/kratoscarf/examples/app/auth-app/internal/conf"
	"github.com/bizjs/kratoscarf/examples/app/auth-app/internal/service"

	authjwt "github.com/bizjs/kratoscarf/auth/jwt"
	"github.com/bizjs/kratoscarf/auth/session"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(c *conf.ServerConfig, authSvc *service.AuthService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			authjwt.Middleware(authSvc.JWTAuth(),
				authjwt.WithSkipPaths(
					"/jwt/login", "/jwt/refresh",
					"/session/login", "/session/logout", "/session/profile",
				),
			),
			session.Middleware(authSvc.SessionMgr(),
				session.WithSkipPaths(
					"/jwt/login", "/jwt/refresh", "/jwt/profile",
					"/session/login",
				),
			),
		),
	}
	if c.HTTP.Network != "" {
		opts = append(opts, http.Network(c.HTTP.Network))
	}
	if c.HTTP.Addr != "" {
		opts = append(opts, http.Address(c.HTTP.Addr))
	}
	if t := c.HTTP.ParseTimeout(); t > 0 {
		opts = append(opts, http.Timeout(t))
	}
	srv := http.NewServer(opts...)

	// Register proto-generated HTTP routes.
	// The generated code uses ctx.Middleware() internally,
	// so JWT/Session middleware context propagation works automatically.
	v1.RegisterAuthServiceHTTPServer(srv, authSvc)

	return srv
}
