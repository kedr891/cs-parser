package v1

import (
	"github.com/cs-parser/internal/usecase"
	"github.com/cs-parser/pkg/logger"
	"github.com/go-playground/validator/v10"
)

// V1 -.
type V1 struct {
	t usecase.Translation
	l logger.Interface
	v *validator.Validate
}
