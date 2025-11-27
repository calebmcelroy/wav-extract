package wav

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Reader struct {
	r          io.Reader
	headerRead bool

	AudioFormat   int
	NumChans      int
	SampleRate    int
	ByteRate      int
	BlockAlign    int
	BitsPerSample int
	DataSize      int
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

func (r *Reader) Read(p []byte) (n int, err error) {
	if !r.headerRead {
		err := r.ReadHeader()
		if err != nil {
			return 0, err
		}
	}

	return r.r.Read(p)
}

func (r *Reader) ReadHeader() error {
	if r.headerRead {
		return nil
	}

	// read RIFF header
	riffHeader := make([]byte, 12)
	_, err := r.r.Read(riffHeader)
	if err != nil {
		return err
	}

	// validate RIFF header
	if string(riffHeader[:4]) != "RIFF" {
		return fmt.Errorf("invalid RIFF header")
	}

	if string(riffHeader[8:12]) != "WAVE" {
		return fmt.Errorf("invalid WAVE header")
	}

	// read fmt chunk header
	fmtHeader := make([]byte, 8)
	if _, err = r.r.Read(fmtHeader); err != nil {
		return err
	}
	if string(fmtHeader[:4]) != "fmt " {
		return fmt.Errorf("invalid fmt header")
	}
	fmtSize := int(binary.LittleEndian.Uint32(fmtHeader[4:8]))
	if fmtSize < 16 {
		return fmt.Errorf("invalid fmt header size")
	}

	// read fmt data (only need first 16 bytes, skip rest)
	fmtData := make([]byte, fmtSize)
	if _, err = r.r.Read(fmtData); err != nil {
		return err
	}
	r.AudioFormat = int(binary.LittleEndian.Uint16(fmtData[0:2]))
	r.NumChans = int(binary.LittleEndian.Uint16(fmtData[2:4]))
	r.SampleRate = int(binary.LittleEndian.Uint32(fmtData[4:8]))
	r.ByteRate = int(binary.LittleEndian.Uint32(fmtData[8:12]))
	r.BlockAlign = int(binary.LittleEndian.Uint16(fmtData[12:14]))
	r.BitsPerSample = int(binary.LittleEndian.Uint16(fmtData[14:16]))

	// read until data chunk
	chunkHeader := make([]byte, 8)
	for {
		_, err = r.r.Read(chunkHeader)
		if err != nil {
			return err
		}

		headerID := string(chunkHeader[:4])
		headerSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		if headerID != "data" {
			// skip chunk
			n, err := r.r.Read(make([]byte, int(headerSize)))
			if err != nil {
				return err
			}

			if n != int(headerSize) {
				return fmt.Errorf("data chunk not found")
			}

			continue
		}

		// found data chunk!
		r.DataSize = int(headerSize)
		break
	}

	r.headerRead = true

	return nil
}
