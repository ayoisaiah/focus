package focus

import "errors"

var (
	errUnableToSaveSession = errors.New("Unable to persist interrupted session")
	errInvalidSoundFormat  = errors.New(
		"Invalid sound file format. Only MP3, OGG, FLAC, and WAV files are supported",
	)
	errNoPausedSession = errors.New(
		"Paused session not found, please start a new session",
	)

	errParsingDate = errors.New(
		"The specified date format must be: YYYY-MM-DD or YYYY-MM-DD HH:MM:SS PM",
	)
	errInvalidDateRange = errors.New(
		"The end date must not be earlier than the start date",
	)

	errFocusRunning = errors.New(
		"Is Focus already running? Only one instance can be active at a time",
	)
)
