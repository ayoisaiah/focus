package config

import "github.com/ayoisaiah/focus/internal/apperr"

var (
	errSessionOverlap = &apperr.Error{
		Message: "new sessions cannot overlap with existing ones",
	}

	errConfigOption = &apperr.Error{
		Message: "config option error",
	}

	errConfigValidation = &apperr.Error{
		Message: "config validation error",
	}

	errReadConfig = &apperr.Error{
		Message: "reading config file failed",
	}

	errWriteConfig = &apperr.Error{
		Message: "writing default config failed",
	}

	errShortBreakTooLong = &apperr.Error{
		Message: "short break duration (%v) must be less than work duration (%v)",
	}

	errLongBreakTooShort = &apperr.Error{
		Message: "long break duration (%v) must be greater than short break duration (%v)",
	}

	errUnknownAlertSound = &apperr.Error{
		Message: "unknown alert sound: %s",
	}

	errUnknownAmbientSound = &apperr.Error{
		Message: "unknown ambient sound: %s",
	}

	errInvalidSoundFormat = &apperr.Error{
		Message: "invalid sound file format: %s (must be mp3, ogg, flac, or wav)",
	}

	errInvalidColor = &apperr.Error{
		Message: "%s color must be a valid hex color code (e.g. #FF0000), got %s",
	}

	errEmptyMsg = &apperr.Error{
		Message: "%s message cannot be empty",
	}

	errInvalidDuration = &apperr.Error{
		Message: "%s duration must be between %v and %v",
	}

	errInvalidLongBreakInterval = &apperr.Error{
		Message: "long break interval must be between %d and %d sessions",
	}
)
