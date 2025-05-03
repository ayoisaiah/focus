package timer

import (
	"io/fs"
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

	err := speaker.Init(
		format.SampleRate,
		format.SampleRate.N(time.Duration(int(time.Second)/DefaultBufferSize)),
	)
	if err != nil {
		return err
	}

	speakerInitialized = true

	return nil
}

// prepSoundStream returns an audio stream for the specified sound.
func prepSoundStream(sound string) (beep.StreamSeekCloser, error) {
	var (
		f      fs.File
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

	defer f.Close()

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
		return nil, errInvalidSoundFormat
	}

	if err != nil {
		return nil, err
	}

	err = initSpeaker(format)
	if err != nil {
		return nil, err
	}

	err = stream.Seek(0)
	if err != nil {
		return nil, err
	}

	return stream, nil
}

func (t *Timer) setAmbientSound() error {
	if t.SoundStream != nil {
		speaker.Clear()
	}

	if t.Opts.Settings.AmbientSound == "" {
		return nil
	}

	stream, err := prepSoundStream(t.Opts.Settings.AmbientSound)
	if err != nil {
		return err
	}

	infiniteStream, err := beep.Loop2(stream)
	if err != nil {
		return err
	}

	t.SoundStream = infiniteStream

	speaker.Play(t.SoundStream)

	return nil
}
