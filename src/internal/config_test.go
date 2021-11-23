package focus

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

func init() {
	configDir = "focus_test"
}

func copyConfig(t *testing.T, src, dest string) error {
	t.Helper()

	original, err := os.Open(src)
	if err != nil {
		return err
	}

	defer original.Close()

	pathToConfigFile, err := xdg.ConfigFile(dest)
	if err != nil {
		return err
	}

	// Create new file
	config, err := os.Create(pathToConfigFile)
	if err != nil {
		return err
	}

	defer config.Close()

	_, err = io.Copy(config, original)
	if err != nil {
		return err
	}

	t.Cleanup(func() {
		err := os.Remove(pathToConfigFile)
		if err != nil {
			t.Logf(
				"Unable to delete test file at: %v due to error: %v",
				pathToConfigFile,
				err,
			)
		}
	})

	return nil
}

func TestConfigFile(t *testing.T) {
	type test struct {
		fileName string
		valid    bool
	}

	testFiles := []test{
		{"valid_config_1.yml", true},
		{"valid_config_2.yml", true},
		{"invalid_config.yml", false},
		{"empty_config.yml", false},
	}

	for _, v := range testFiles {
		configFileName = v.fileName

		srcPath := filepath.Join("..", "..", "testdata", v.fileName)
		destPath := filepath.Join(configDir, configFileName)

		err := copyConfig(t, srcPath, destPath)
		if err != nil {
			t.Fatalf(
				"[%s]: Error occurred while calling copyConfig: %v",
				v.fileName,
				err,
			)
		}

		got, err := NewConfig()
		if err != nil {
			t.Fatalf(
				"[%s]: Error occurred while calling NewConfig: %v",
				v.fileName,
				err,
			)
		}

		expected := &Config{}

		if !v.valid {
			expected.defaults()
		} else {
			b, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("[%s]: Error occurred while reading the source config file: %v", v.fileName, err)
			}

			err = yaml.Unmarshal(b, expected)
			if err != nil {
				t.Fatalf("[%s]: Error occurred while unmarshalling the yaml source config: %v", v.fileName, err)
			}
		}

		if !cmp.Equal(got, expected) {
			t.Fatalf(
				"[%s]: Incorrect config. Expected: %+v, but got: %+v",
				v.fileName,
				expected,
				got,
			)
		}
	}
}
