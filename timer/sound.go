package timer

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"

	"github.com/ayoisaiah/focus/config"
	"github.com/ayoisaiah/focus/internal/pathutil"
	"github.com/ayoisaiah/focus/internal/static"
)

var soundOpts []string

func init() {
	dir, err := os.ReadDir(filepath.Join(xdg.DataHome, config.Dir(), "static"))
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range dir {
		soundOpts = append(soundOpts, pathutil.StripExtension(v.Name()))
	}
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

		f, err = static.Files.Open(static.FilePath(sound))
		if err != nil {
			// TODO: Update error
			return nil, err
		}
	} else {
		f, err = os.Open(sound)
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

	bufferSize := 10

	err = speaker.Init(
		format.SampleRate,
		format.SampleRate.N(time.Duration(int(time.Second)/bufferSize)),
	)
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
	var infiniteStream beep.Streamer

	if t.Opts.AmbientSound != "" {
		stream, err := prepSoundStream(t.Opts.AmbientSound)
		if err != nil {
			return err
		}

		infiniteStream = beep.Loop(-1, stream)
	}

	t.SoundStream = infiniteStream

	return nil
}
