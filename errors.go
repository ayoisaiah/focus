package focus

type Error string

func (e Error) Error() string { return string(e) }

const (
	errUnableToSaveSession = Error("Unable to persist interrupted session")
	errInvalidSoundFormat  = Error(
		"Invalid sound file format. Only MP3, OGG, FLAC, and WAV files are supported",
	)
	errNoPausedSession = Error(
		"Existing paused session was not detected. Please start a new session",
	)

	errReadingInput = Error(
		"An error occurred while reading input. Please try again",
	)
	errExpectedInteger = Error(
		"Expected an integer that must be greater than zero",
	)
	errInitFailed = Error(
		"Unable to initialise Focus settings from the configuration file",
	)

	errParsingDate = Error(
		"The specified date format must be: YYYY-MM-DD or YYYY-MM-DD HH:MM:SS PM",
	)
	errInvalidDateRange = Error(
		"The end date must not be earlier than the start date",
	)

	errFocusRunning = Error(
		"Is Focus already running? Only one instance can be active at a time",
	)
)
