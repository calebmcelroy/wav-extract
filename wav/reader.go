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

	// read fmt header
	fmtData := make([]byte, 24)
	_, err = r.r.Read(fmtData)

	if err != nil {
		return err
	}

	// validate fmt header
	if string(fmtData[:4]) != "fmt " {
		return fmt.Errorf("invalid fmt header")
	}

	// check size of fmt header
	if binary.LittleEndian.Uint32(fmtData[4:8]) != 16 {
		return fmt.Errorf("invalid fmt header size")
	}

	// read fmt header data (values are little-endian)
	r.AudioFormat = int(binary.LittleEndian.Uint16(fmtData[8:10]))
	r.NumChans = int(binary.LittleEndian.Uint16(fmtData[10:12]))
	r.SampleRate = int(binary.LittleEndian.Uint32(fmtData[12:16]))
	r.ByteRate = int(binary.LittleEndian.Uint32(fmtData[16:20]))
	r.BlockAlign = int(binary.LittleEndian.Uint16(fmtData[20:22]))
	r.BitsPerSample = int(binary.LittleEndian.Uint16(fmtData[22:24]))

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
