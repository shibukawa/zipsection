package zipsection

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var testCases = []struct {
	filename string
	osname   string
	size     int64
}{
	{"testexe.exe", "windows(PE)", 575836},
	{"testexe.linux", "linux(elf)", 575836},
	{"testexe.darwin", "darwin(mach-o)", 575836},
}

func TestDetect(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Read zip section from %s binary", tc.osname), func(t *testing.T) {
			cwd, _ := os.Getwd()
			sectionSize, err := Detect(filepath.Join(cwd, "testdata", tc.filename))
			if err != nil {
				t.Error("err should be nil, but", err)
			}
			if sectionSize != tc.size {
				t.Errorf("zipReader should be %d bytes, but %d bytes", tc.size, sectionSize)
			}
		})
	}
}

func TestOpen(t *testing.T) {
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Read zip section from %s binary", tc.osname), func(t *testing.T) {
			cwd, _ := os.Getwd()

			zipReader, closer, err := Open(filepath.Join(cwd, "testdata", tc.filename))
			if closer != nil {
				defer closer.Close()
			}
			if err != nil {
				t.Error("err should be nil, but", err)
			}
			if zipReader == nil {
				t.Error("zipReader should not be nil")
			}
		})
	}
}
