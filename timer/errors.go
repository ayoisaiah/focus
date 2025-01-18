package timer

import "github.com/ayoisaiah/focus/internal/apperr"

var (
	errInvalidSoundFormat = &apperr.Error{
		Message: "sound file must be in mp3, ogg, flac, or wav format",
	}

	errInvalidInput = &apperr.Error{
		Message: "invalid input: only comma-separated numbers are accepted",
	}

	errSessionOverlap = &apperr.Error{
		Message: "new sessions cannot overlap with existing ones",
	}
)
