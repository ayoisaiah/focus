package timer

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"

	"github.com/ayoisaiah/focus/internal/config"
)

// DefaultBufferSize controls audio buffering.
const DefaultBufferSize = 10

var speakerInitialized bool

func initSpeaker(format beep.Format) error {
	if speakerInitialized {
		return nil
	}

	// Initialize speaker with a larger buffer to prevent audio glitches
	err := speaker.Init(
		format.SampleRate,
		format.SampleRate.N(time.Duration(int(time.Second)/5)), // Larger buffer
	)
	if err != nil {
		return err
	}

	speakerInitialized = true

	return nil
}

// prepAlertSoundStream returns an audio stream for alert/notification sounds.
func prepAlertSoundStream(sound string) (beep.StreamSeekCloser, error) {
	
	var (
		f      *os.File
		err    error
		stream beep.StreamSeekCloser
		format beep.Format
	)

	ext := filepath.Ext(sound)
	if ext == "" {
		sound += ".ogg"
	}

	soundPath := filepath.Join(config.AlertSoundPath(), sound)
	f, err = os.Open(soundPath)
	if err != nil {
		return nil, err
	}

	ext = filepath.Ext(sound)

	switch ext {
	case ".ogg":
		stream, format, err = vorbis.Decode(f)
	case ".mp3":
		stream, format, err = mp3.Decode(f)
	case ".flac":
		stream, format, err = flac.Decode(f)
	case ".wav":
		stream, format, err = wav.Decode(f)
	default:
		f.Close()
		return nil, errInvalidSoundFormat
	}

	if err != nil {
		f.Close()
		return nil, err
	}

	err = initSpeaker(format)
	if err != nil {
		f.Close()
		return nil, err
	}

	err = stream.Seek(0)
	if err != nil {
		f.Close()
		return nil, err
	}

	return &fileStreamWrapper{
		StreamSeekCloser: stream,
		file:             f,
	}, nil
}

// prepSoundStream returns an audio stream for the specified sound.
// For one-time sounds, the file will be closed when the stream is closed.
// For ambient sounds, the file must remain open for continuous playback.
func prepSoundStream(sound string) (beep.StreamSeekCloser, error) {
	var (
		f      *os.File
		err    error
		stream beep.StreamSeekCloser
		format beep.Format
	)

	ext := filepath.Ext(sound)
	if ext == "" {
		sound += ".ogg"
	}

	f, err = os.Open(
		filepath.Join(config.AmbientSoundPath(), sound),
	)
	if err != nil {
		return nil, err
	}

	ext = filepath.Ext(sound)

	switch ext {
	case ".ogg":
		stream, format, err = vorbis.Decode(f)
	case ".mp3":
		stream, format, err = mp3.Decode(f)
	case ".flac":
		stream, format, err = flac.Decode(f)
	case ".wav":
		stream, format, err = wav.Decode(f)
	default:
		f.Close()
		return nil, errInvalidSoundFormat
	}

	if err != nil {
		f.Close()
		return nil, err
	}

	err = initSpeaker(format)
	if err != nil {
		f.Close()
		return nil, err
	}

	err = stream.Seek(0)
	if err != nil {
		f.Close()
		return nil, err
	}

	// Create a wrapper that closes the file when the stream is closed
	return &fileStreamWrapper{
		StreamSeekCloser: stream,
		file:             f,
	}, nil
}

// fileStreamWrapper wraps a stream and ensures the file is closed when the stream is closed
type fileStreamWrapper struct {
	beep.StreamSeekCloser
	file *os.File
}

func (w *fileStreamWrapper) Close() error {
	streamErr := w.StreamSeekCloser.Close()
	fileErr := w.file.Close()
	
	if streamErr != nil {
		return streamErr
	}
	return fileErr
}

func (t *Timer) setAmbientSound() error {
	// Preserve timer running state
	wasRunning := t.clock.Running()
	
	// If turning off sound, clear everything
	if t.Opts.Settings.AmbientSound == "" || t.Opts.Settings.AmbientSound == "off" {
		if t.SoundStream != nil {
			speaker.Clear()
			if closer, ok := t.SoundStream.(interface{ Close() error }); ok {
				closer.Close()
			}
			t.SoundStream = nil
		}
		return nil
	}

	// Close existing stream but be careful with speaker operations
	if t.SoundStream != nil {
		if closer, ok := t.SoundStream.(interface{ Close() error }); ok {
			closer.Close()
		}
		t.SoundStream = nil
	}

	// Clear speaker - this might be affecting timer state
	speaker.Clear()

	stream, err := prepSoundStream(t.Opts.Settings.AmbientSound)
	if err != nil {
		return err
	}

	infiniteStream := beep.Loop(-1, stream)
	t.SoundStream = infiniteStream

	speaker.Play(t.SoundStream)
	
	// Try to restore timer state if it got affected
	if wasRunning && !t.clock.Running() {
		// The timer stopped when it shouldn't have, try to restart it
		t.clock.Toggle()
	}

	return nil
}
