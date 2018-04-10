package zipsection

// Codes in this file are from archive/zip package of golang

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

var (
	ErrFormat    = errors.New("zip: not a valid zip file")
	ErrAlgorithm = errors.New("zip: unsupported compression algorithm")
	ErrChecksum  = errors.New("zip: checksum error")
)

const (
	directoryEndLen         = 22
	directory64LocLen       = 20
	directory64EndLen       = 56
	directory64LocSignature = 0x07064b50
	directory64EndSignature = 0x06064b50
)

type readBuf []byte

func (b *readBuf) uint8() uint8 {
	v := (*b)[0]
	*b = (*b)[1:]
	return v
}

func (b *readBuf) uint16() uint16 {
	v := binary.LittleEndian.Uint16(*b)
	*b = (*b)[2:]
	return v
}

func (b *readBuf) uint32() uint32 {
	v := binary.LittleEndian.Uint32(*b)
	*b = (*b)[4:]
	return v
}

func (b *readBuf) uint64() uint64 {
	v := binary.LittleEndian.Uint64(*b)
	*b = (*b)[8:]
	return v
}

func (b *readBuf) sub(n int) readBuf {
	b2 := (*b)[:n]
	*b = (*b)[n:]
	return b2
}

func findSignatureInBlock(b []byte) int {
	for i := len(b) - directoryEndLen; i >= 0; i-- {
		// defined from directoryEndSignature in struct.go
		if b[i] == 'P' && b[i+1] == 'K' && b[i+2] == 0x05 && b[i+3] == 0x06 {
			// n is length of comment
			n := int(b[i+directoryEndLen-2]) | int(b[i+directoryEndLen-1])<<8
			if n+directoryEndLen+i <= len(b) {
				return i
			}
		}
	}
	return -1
}

func readDirectoryEnd(r io.ReaderAt, size int64) (int64, error) {
	// look for directoryEndSignature in the last 1k, then in the last 65k
	var buf []byte
	var directoryEndOffset int64
	for i, bLen := range []int64{1024, 65 * 1024} {
		if bLen > size {
			bLen = size
		}
		buf = make([]byte, int(bLen))
		if _, err := r.ReadAt(buf, size-bLen); err != nil && err != io.EOF {
			return -1, err
		}
		if p := findSignatureInBlock(buf); p >= 0 {
			buf = buf[p:]
			directoryEndOffset = size - bLen + int64(p)
			break
		}
		if i == 1 || bLen == size {
			return -1, ErrFormat
		}
	}

	// read header into struct
	b := readBuf(buf[10:]) // skip signature[4], diskNbr[2], dirDiskNbr[2], dirRecordsThisDisk[2]

	directoryRecords := uint64(b.uint16())
	directorySize := uint64(b.uint32())
	directoryOffset := uint64(b.uint32())

	// These values mean that the file can be a zip64 file
	if directoryRecords == 0xffff || directorySize == 0xffff || directoryOffset == 0xffffffff {
		p, err := findDirectory64End(r, directoryEndOffset)
		if err == nil && p >= 0 {
			directoryOffset, err = readDirectory64End(r, p)
		}
		if err != nil {
			return -1, err
		}
	}
	// Make sure directoryOffset points to somewhere in our file.
	if o := int64(directoryOffset); o < 0 || o >= size {
		return -1, ErrFormat
	}
	return int64(directorySize+directoryOffset) + (size - directoryEndOffset), nil
}

func findDirectory64End(r io.ReaderAt, directoryEndOffset int64) (int64, error) {
	locOffset := directoryEndOffset - directory64LocLen
	if locOffset < 0 {
		return -1, nil // no need to look for a header outside the file
	}
	buf := make([]byte, directory64LocLen)
	if _, err := r.ReadAt(buf, locOffset); err != nil {
		return -1, err
	}
	b := readBuf(buf)
	if sig := b.uint32(); sig != directory64LocSignature {
		return -1, nil
	}
	if b.uint32() != 0 { // number of the disk with the start of the zip64 end of central directory
		return -1, nil // the file is not a valid zip64-file
	}
	p := b.uint64()      // relative offset of the zip64 end of central directory record
	if b.uint32() != 1 { // total number of disks
		return -1, nil // the file is not a valid zip64-file
	}
	return int64(p), nil
}

// readDirectory64End reads the zip64 directory end and updates the
// directory end with the zip64 directory end values.
func readDirectory64End(r io.ReaderAt, offset int64) (sectionSize uint64, err error) {
	buf := make([]byte, directory64EndLen)
	if _, err := r.ReadAt(buf, offset); err != nil {
		return math.MaxUint64, err
	}

	b := readBuf(buf)
	if sig := b.uint32(); sig != directory64EndSignature {
		return math.MaxUint64, ErrFormat
	}

	b = b[12+4*2+8*2:] // skip dir size, version and version needed (uint64 + 2x uint16)
	// diskNbr[4], dirDiskNbr[4], dirRecordsThisDisk[8], directoryRecords[8]
	// directorySize[8]
	directorySize := b.uint64()
	directoryOffset := b.uint64() // offset of start of central directory with respect to the starting disk number
	return directorySize + directoryOffset + directory64EndLen, nil
}
