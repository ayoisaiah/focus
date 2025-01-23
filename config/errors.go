package config

import "github.com/ayoisaiah/focus/internal/apperr"

var errSessionOverlap = &apperr.Error{
	Message: "new sessions cannot overlap with existing ones",
}
