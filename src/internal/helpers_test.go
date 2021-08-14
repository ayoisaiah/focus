package focus

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"
)

var (
	update = flag.Bool("update", false, "used to update golden files in tests")
)

func goldenFile(
	t *testing.T,
	filename, content string,
	shouldUpdate bool,
) string {
	t.Helper()

	filePath := filepath.Join("..", "..", "testdata", filename)

	// create or open the file
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		t.Fatal(err)
	}

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}

	// If update flag is set or file is empty
	if shouldUpdate || fileInfo.Size() == 0 {
		err = file.Truncate(0)
		if err != nil {
			t.Fatal(err)
		}

		_, err = file.Seek(0, 0)
		if err != nil {
			t.Fatal(err)
		}

		_, err = file.WriteString(content)
		if err != nil {
			t.Fatal(err)
		}

		return content
	}

	b, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
