package server

import (
	"github.com/bizjs/kratoscarf/examples/app/router-validation-demo/internal/conf"
	"github.com/bizjs/kratoscarf/examples/app/router-validation-demo/internal/service"
	"github.com/bizjs/kratoscarf/response"
	"github.com/bizjs/kratoscarf/router"
	"github.com/bizjs/kratoscarf/validation"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(c *conf.ServerConfig, todoSvc *service.TodoService) *kratoshttp.Server {
	httpSrv := kratoshttp.NewServer(
		kratoshttp.Address(c.HTTP.Addr),
		kratoshttp.ResponseEncoder(response.NewHTTPResponseEncoder()),
		kratoshttp.ErrorEncoder(response.NewHTTPErrorEncoder()),
	)

	r := router.NewRouter(httpSrv,
		router.WithValidator(validation.New()),    // Bind() auto-validates
		router.WithResponseWrapper(response.Wrap), // Success() auto-wraps {code, message, data}
	)
	todoSvc.RegisterRoutes(r)

	return httpSrv
}
