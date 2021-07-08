package focus

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gookit/color"
)

type colorString string

const (
	red    colorString = "red"
	green  colorString = "green"
	yellow colorString = "yellow"
)

func printColor(c colorString, text string) string {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return text
	}

	switch c {
	case yellow:
		return color.HEX("#FFAB00").Sprint(text)
	case green:
		return color.HEX("#23D160").Sprint(text)
	case red:
		return color.HEX("#FF2F2F").Sprint(text)
	}

	return text
}

func saveToDisk(val interface{}, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		ferr := file.Close()
		if ferr != nil {
			err = ferr
		}
	}()

	writer := bufio.NewWriter(file)

	b, err := json.MarshalIndent(val, "", "    ")
	if err != nil {
		return err
	}

	_, err = writer.Write(b)
	if err != nil {
		return err
	}

	return writer.Flush()
}

func retrieveFromDisk(filename string) ([]byte, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(filepath.Join(dir, configFolder), 0750)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, configFolder, filename)

	return os.ReadFile(path)
}

func numberPrompt(reader *bufio.Reader, defaultVal int) (int, error) {
	input, err := reader.ReadString('\n')
	if err != nil {
		return 0, errors.New(errReadingInput)
	}

	reader.Reset(os.Stdin)

	input = strings.TrimSpace(strings.TrimSuffix(input, "\n"))
	if input == "" {
		return defaultVal, nil
	}

	num, err := strconv.Atoi(input)
	if err != nil {
		return 0, errors.New(errExpectedNumber)
	}

	if num <= 0 {
		return 0, errors.New(errExpectPositiveInteger)
	}

	return num, nil
}

func stringPrompt(reader *bufio.Reader, defaultVal string) (string, error) {
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", errors.New(errReadingInput)
	}

	reader.Reset(os.Stdin)

	input = strings.TrimSpace(strings.TrimSuffix(input, "\n"))
	if input == "" {
		input = defaultVal
	}

	return input, nil
}
