package zipdetect

import (
	"archive/zip"
	"io"
	"os"
)

func Detect(path string) (sectionSize int64, err error) {
	file, err := os.Open(path)
	if err != nil {
		return -1, err
	}
	defer file.Close()
	finfo, err := file.Stat()
	if err != nil {
		return -1, err
	}
	return DetectFromReader(file, finfo.Size())
}

func DetectFromReader(reader io.ReaderAt, size int64) (zipSectionSize int64, err error) {
	return readDirectoryEnd(reader, size)
}

func Open(path string) (*zip.Reader, io.Closer, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	finfo, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	reader, err := OpenFromReader(file, finfo.Size())
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	return reader, file, nil
}

func OpenFromReader(reader io.ReaderAt, size int64) (*zip.Reader, error) {
	sectionSize, err := DetectFromReader(reader, size)
	if err != nil {
		return nil, err
	}
	section := io.NewSectionReader(reader, size-sectionSize, sectionSize)
	return zip.NewReader(section, section.Size())
}
