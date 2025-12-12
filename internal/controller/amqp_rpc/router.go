package v1

import (
	v1 "github.com/cs-parser/internal/controller/amqp_rpc/v1"
	"github.com/cs-parser/internal/usecase"
	"github.com/cs-parser/pkg/logger"
	"github.com/cs-parser/pkg/rabbitmq/rmq_rpc/server"
)

// NewRouter -.
func NewRouter(t usecase.Translation, l logger.Interface) map[string]server.CallHandler {
	routes := make(map[string]server.CallHandler)

	{
		v1.NewTranslationRoutes(routes, t, l)
	}

	return routes
}
