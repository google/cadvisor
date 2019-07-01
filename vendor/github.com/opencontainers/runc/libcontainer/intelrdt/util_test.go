// +build linux

/*
 * Utility for testing Intel RDT operations.
 * Creates a mock of the Intel RDT "resource control" filesystem for the duration of the test.
 */
package intelrdt

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/opencontainers/runc/libcontainer/configs"
)

type intelRdtTestUtil struct {
	// intelRdt data to use in tests
	IntelRdtData *intelRdtData

	// Path to the mock Intel RDT "resource control" filesystem directory
	IntelRdtPath string

	// Temporary directory to store mock Intel RDT "resource control" filesystem
	tempDir string
	t       *testing.T
}

// Creates a new test util
func NewIntelRdtTestUtil(t *testing.T) *intelRdtTestUtil {
	d := &intelRdtData{
		config: &configs.Config{
			IntelRdt: &configs.IntelRdt{},
		},
	}
	tempDir, err := ioutil.TempDir("", "intelrdt_test")
	if err != nil {
		t.Fatal(err)
	}
	d.root = tempDir
	testIntelRdtPath := filepath.Join(d.root, "resctrl")
	if err != nil {
		t.Fatal(err)
	}

	// Ensure the full mock Intel RDT "resource control" filesystem path exists
	err = os.MkdirAll(testIntelRdtPath, 0755)
	if err != nil {
		t.Fatal(err)
	}
	return &intelRdtTestUtil{IntelRdtData: d, IntelRdtPath: testIntelRdtPath, tempDir: tempDir, t: t}
}

func (c *intelRdtTestUtil) cleanup() {
	os.RemoveAll(c.tempDir)
}

// Write the specified contents on the mock of the specified Intel RDT "resource control" files
func (c *intelRdtTestUtil) writeFileContents(fileContents map[string]string) {
	for file, contents := range fileContents {
		err := writeFile(c.IntelRdtPath, file, contents)
		if err != nil {
			c.t.Fatal(err)
		}
	}
}
