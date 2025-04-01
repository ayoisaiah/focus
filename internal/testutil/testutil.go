package testutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sebdah/goldie/v2"

	"github.com/ayoisaiah/focus/internal/osutil"
)

type GoldenTest interface {
	Output() ([]byte, string)
}

// CompareGoldenFile verifies that the output of an operation matches
// the expected output.
func CompareGoldenFile(t *testing.T, tc GoldenTest) {
	t.Helper()

	if runtime.GOOS == osutil.Windows {
		// TODO: need to sort out line endings
		t.Skip("skipping golden file test in Windows")
	}

	g := goldie.New(
		t,
		goldie.WithFixtureDir("testdata"),
	)

	compareOutput := func(output []byte, goldenFileName string) {
		if output != nil {
			g.Assert(t, goldenFileName, output)
		} else {
			f := filepath.Join("testdata", goldenFileName+".golden")
			if _, err := os.Stat(f); err == nil || errors.Is(err, os.ErrExist) {
				t.Fatalf("expected no output, but golden file exists: %s", f)
			}
		}
	}

	snap, golden := tc.Output()

	compareOutput(snap, golden)
}

func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return nil
}
