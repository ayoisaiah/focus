package timer

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"

	"github.com/ayoisaiah/focus/internal/pathutil"
	"github.com/ayoisaiah/focus/internal/static"
)

// DefaultBufferSize controls audio buffering.
const DefaultBufferSize = 10

var soundOpts []string

var speakerInitialized bool

func init() {
	dir, err := fs.ReadDir(
		static.Files,
		filepath.Join("files", "ambient_sound"),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range dir {
		soundOpts = append(soundOpts, pathutil.StripExtension(v.Name()))
	}

	path := filepath.Join(xdg.DataHome, "focus", "ambient_sound")

	dirs, err := os.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range dirs {
		soundOpts = append(soundOpts, v.Name())
	}
}

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
	// without extension, treat as OGG file
	if ext == "" {
		sound += ".ogg"

		f, err = static.Files.Open(static.AmbientSound(sound))
		if err != nil {
			// TODO: Update error
			return nil, err
		}
	} else {
		f, err = os.Open(filepath.Join(xdg.DataHome, "focus", "ambient_sound", sound))
		// TODO: Update error
		if err != nil {
			return nil, err
		}
	}

	defer func() {
		_ = f.Close()
	}()

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

	if t.Opts.Sound.AmbientSound == "" {
		return nil
	}

	stream, err := prepSoundStream(t.Opts.Sound.AmbientSound)
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
