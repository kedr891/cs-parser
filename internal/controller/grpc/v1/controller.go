package v1

import (
	"github.com/go-playground/validator/v10"
	v1 "github.com/kedr891/cs-parser/docs/proto/v1"
	"github.com/kedr891/cs-parser/internal/usecase"
	"github.com/kedr891/cs-parser/pkg/logger"
)

// V1 -.
type V1 struct {
	v1.TranslationServer

	t usecase.Translation
	l logger.Interface
	v *validator.Validate
}
