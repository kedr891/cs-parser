package v1

import (
	"github.com/go-playground/validator/v10"
	v1 "github.com/kedr891/cs-parser/docs/proto/v1"
	"github.com/kedr891/cs-parser/internal/usecase"
	"github.com/kedr891/cs-parser/pkg/logger"
	pbgrpc "google.golang.org/grpc"
)

// NewTranslationRoutes -.
func NewTranslationRoutes(app *pbgrpc.Server, t usecase.Translation, l logger.Interface) {
	r := &V1{t: t, l: l, v: validator.New(validator.WithRequiredStructEnabled())}

	{
		v1.RegisterTranslationServer(app, r)
	}
}
