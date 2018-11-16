package godirwalk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/karrick/godirwalk"
)

const testScratchBufferSize = 16 * 1024

func helperFilepathWalk(tb testing.TB, osDirname string) []string {
	var entries []string
	err := filepath.Walk(osDirname, func(osPathname string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "skip" {
			return filepath.SkipDir
		}
		// filepath.Walk invokes callback function with a slashed version of the
		// pathname, while godirwalk invokes callback function with the
		// os-specific pathname separator.
		entries = append(entries, filepath.ToSlash(osPathname))
		return nil
	})
	if err != nil {
		tb.Fatal(err)
	}
	return entries
}

func helperGodirwalkWalk(tb testing.TB, osDirname string) []string {
	var entries []string
	err := godirwalk.Walk(osDirname, &godirwalk.Options{
		Callback: func(osPathname string, dirent *godirwalk.Dirent) error {
			if dirent.Name() == "skip" {
				return filepath.SkipDir
			}
			// filepath.Walk invokes callback function with a slashed version of
			// the pathname, while godirwalk invokes callback function with the
			// os-specific pathname separator.
			entries = append(entries, filepath.ToSlash(osPathname))
			return nil
		},
		ScratchBuffer: make([]byte, testScratchBufferSize),
	})
	if err != nil {
		tb.Fatal(err)
	}
	return entries
}

func symlinkAbs(oldname, newname string) error {
	absDir, err := filepath.Abs(oldname)
	if err != nil {
		return err
	}
	return os.Symlink(absDir, newname)
}

func TestWalkSkipDir(t *testing.T) {
	// Ensure the results from calling filepath.Walk exactly match the results
	// for calling this library's walk function.

	test := func(t *testing.T, osDirname string) {
		expected := helperFilepathWalk(t, osDirname)
		actual := helperGodirwalkWalk(t, osDirname)

		if got, want := len(actual), len(expected); got != want {
			t.Fatalf("\n(GOT)\n\t%#v\n(WNT)\n\t%#v", actual, expected)
		}

		for i := 0; i < len(actual); i++ {
			if got, want := actual[i], expected[i]; got != want {
				t.Errorf("(GOT) %v; (WNT) %v", got, want)
			}
		}
	}

	// Test cases for encountering the filepath.SkipDir error at different times
	// from the call.

	t.Run("SkipFileAtRoot", func(t *testing.T) {
		test(t, "testdata/dir1/dir1a")
	})

	t.Run("SkipFileUnderRoot", func(t *testing.T) {
		test(t, "testdata/dir1")
	})

	t.Run("SkipDirAtRoot", func(t *testing.T) {
		test(t, "testdata/dir2/skip")
	})

	t.Run("SkipDirUnderRoot", func(t *testing.T) {
		test(t, "testdata/dir2")
	})

	t.Run("SkipDirOnSymlink", func(t *testing.T) {
		osDirname := "testdata/dir3"
		actual := helperGodirwalkWalk(t, osDirname)

		expected := []string{
			"testdata/dir3",
			"testdata/dir3/aaa.txt",
			"testdata/dir3/zzz",
			"testdata/dir3/zzz/aaa.txt",
		}

		if got, want := len(actual), len(expected); got != want {
			t.Fatalf("\n(GOT)\n\t%#v\n(WNT)\n\t%#v", actual, expected)
		}

		for i := 0; i < len(actual); i++ {
			if got, want := actual[i], expected[i]; got != want {
				t.Errorf("(GOT) %v; (WNT) %v", got, want)
			}
		}
	})
}

func TestWalkFollowSymbolicLinksFalse(t *testing.T) {
	const (
		osDirname = "testdata/dir4"
		symlink   = "testdata/dir4/symlinkToAbsDirectory"
	)

	if err := symlinkAbs("testdata/dir4/zzz", symlink); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Remove(symlink); err != nil {
			t.Error(err)
		}
	}()

	var actual []string
	err := godirwalk.Walk(osDirname, &godirwalk.Options{
		Callback: func(osPathname string, dirent *godirwalk.Dirent) error {
			if dirent.Name() == "skip" {
				return filepath.SkipDir
			}
			// filepath.Walk invokes callback function with a slashed version of
			// the pathname, while godirwalk invokes callback function with the
			// os-specific pathname separator.
			actual = append(actual, filepath.ToSlash(osPathname))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"testdata/dir4",
		"testdata/dir4/aaa.txt",
		"testdata/dir4/symlinkToAbsDirectory",
		"testdata/dir4/symlinkToDirectory",
		"testdata/dir4/symlinkToFile",
		"testdata/dir4/zzz",
		"testdata/dir4/zzz/aaa.txt",
	}

	if got, want := len(actual), len(expected); got != want {
		t.Fatalf("\n(GOT)\n\t%#v\n(WNT)\n\t%#v", actual, expected)
	}

	for i := 0; i < len(actual); i++ {
		if got, want := actual[i], expected[i]; got != want {
			t.Errorf("(GOT) %v; (WNT) %v", got, want)
		}
	}
}

func TestWalkFollowSymbolicLinksTrue(t *testing.T) {
	const (
		osDirname = "testdata/dir4"
		symlink   = "testdata/dir4/symlinkToAbsDirectory"
	)

	if err := symlinkAbs("testdata/dir4/zzz", symlink); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.Remove(symlink); err != nil {
			t.Error(err)
		}
	}()

	var actual []string
	err := godirwalk.Walk(osDirname, &godirwalk.Options{
		FollowSymbolicLinks: true,
		Callback: func(osPathname string, dirent *godirwalk.Dirent) error {
			if dirent.Name() == "skip" {
				return filepath.SkipDir
			}
			// filepath.Walk invokes callback function with a slashed version of
			// the pathname, while godirwalk invokes callback function with the
			// os-specific pathname separator.
			actual = append(actual, filepath.ToSlash(osPathname))
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"testdata/dir4",
		"testdata/dir4/aaa.txt",
		"testdata/dir4/symlinkToAbsDirectory",
		"testdata/dir4/symlinkToAbsDirectory/aaa.txt",
		"testdata/dir4/symlinkToDirectory",
		"testdata/dir4/symlinkToDirectory/aaa.txt",
		"testdata/dir4/symlinkToFile",
		"testdata/dir4/zzz",
		"testdata/dir4/zzz/aaa.txt",
	}

	if got, want := len(actual), len(expected); got != want {
		t.Fatalf("\n(GOT)\n\t%#v\n(WNT)\n\t%#v", actual, expected)
	}

	for i := 0; i < len(actual); i++ {
		if got, want := actual[i], expected[i]; got != want {
			t.Errorf("(GOT) %v; (WNT) %v", got, want)
		}
	}
}

func TestPostChildrenCallback(t *testing.T) {
	const osDirname = "testdata/dir5"

	var actual []string

	err := godirwalk.Walk(osDirname, &godirwalk.Options{
		ScratchBuffer: make([]byte, testScratchBufferSize),
		Callback: func(osPathname string, _ *godirwalk.Dirent) error {
			t.Logf("walk in: %s", osPathname)
			return nil
		},
		PostChildrenCallback: func(osPathname string, de *godirwalk.Dirent) error {
			t.Logf("walk out: %s", osPathname)
			actual = append(actual, osPathname)
			return nil
		},
	})
	if err != nil {
		t.Errorf("(GOT): %v; (WNT): %v", err, nil)
	}

	expected := []string{
		"testdata/dir5/a2/a2a",
		"testdata/dir5/a2",
		"testdata/dir5",
	}

	if got, want := len(actual), len(expected); got != want {
		t.Errorf("(GOT) %v; (WNT) %v", got, want)
	}

	for i := 0; i < len(actual); i++ {
		if i >= len(expected) {
			t.Fatalf("(GOT) %v; (WNT): %v", actual[i], nil)
		}
		if got, want := actual[i], expected[i]; got != want {
			t.Errorf("(GOT) %v; (WNT) %v", got, want)
		}
	}
}

var goPrefix = filepath.Join(os.Getenv("GOPATH"), "src")

func BenchmarkFilepathWalk(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark using user's Go source directory")
	}

	for i := 0; i < b.N; i++ {
		_ = helperFilepathWalk(b, goPrefix)
	}
}

func BenchmarkGoDirWalk(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark using user's Go source directory")
	}

	for i := 0; i < b.N; i++ {
		_ = helperGodirwalkWalk(b, goPrefix)
	}
}

const flameIterations = 10

func BenchmarkFlameGraphFilepathWalk(b *testing.B) {
	for i := 0; i < flameIterations; i++ {
		_ = helperFilepathWalk(b, goPrefix)
	}
}

func BenchmarkFlameGraphGoDirWalk(b *testing.B) {
	for i := 0; i < flameIterations; i++ {
		_ = helperGodirwalkWalk(b, goPrefix)
	}
}
