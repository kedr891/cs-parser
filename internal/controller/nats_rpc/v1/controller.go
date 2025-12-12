package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/kedr891/cs-parser/internal/usecase"
	"github.com/kedr891/cs-parser/pkg/logger"
)

// V1 -.
type V1 struct {
	t usecase.Translation
	l logger.Interface
	v *validator.Validate
}
