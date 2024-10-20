package wav

import (
	"encoding/binary"
	"io"
	"sync/atomic"
)

type Writer struct {
	w             io.WriterAt
	headerWritten bool
	audioFormat   int
	numChans      int
	sampleRate    int
	bitsPerSample int

	dataSize *atomic.Uint32
}

func NewWriter(w io.WriterAt, audioFormat int, numChans int, sampleRate int, bitsPerSample int) *Writer {
	return &Writer{
		w:             w,
		audioFormat:   audioFormat,
		numChans:      numChans,
		sampleRate:    sampleRate,
		bitsPerSample: bitsPerSample,
		dataSize:      &atomic.Uint32{},
	}
}

func (w *Writer) WriteAt(p []byte, off int64) (n int, err error) {
	if !w.headerWritten {
		err := w.writeHeader()
		if err != nil {
			return 0, err
		}
	}

	n, err = w.w.WriteAt(p, off+44)

	if err != nil {
		return 0, err
	}

	w.dataSize.Add(uint32(n))

	return n, nil
}

func (w *Writer) writeHeader() error {
	header := make([]byte, 44)

	// RIFF header
	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], 0)
	copy(header[8:], "WAVE")

	// fmt header
	copy(header[12:], "fmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], uint16(w.numChans))
	binary.LittleEndian.PutUint32(header[24:], uint32(w.sampleRate))
	byteRate := w.sampleRate * w.numChans * w.bitsPerSample / 8
	binary.LittleEndian.PutUint32(header[28:], uint32(byteRate))
	blockAlign := w.numChans * w.bitsPerSample / 8
	binary.LittleEndian.PutUint16(header[32:], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:], uint16(w.bitsPerSample))

	// data header
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], 0)

	_, err := w.w.WriteAt(header, 0)
	if err != nil {
		return err
	}

	return nil
}

func (w *Writer) Close() error {
	if !w.headerWritten {
		err := w.writeHeader()
		if err != nil {
			return err
		}
	}

	// update RIFF header with file size
	fileSizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(fileSizeBytes, w.dataSize.Load()+36)
	_, err := w.w.WriteAt(fileSizeBytes, 4)
	if err != nil {
		return err
	}

	// update data header with data size
	dataSizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(dataSizeBytes, w.dataSize.Load())
	_, err = w.w.WriteAt(dataSizeBytes, 40)
	if err != nil {
		return err
	}

	return nil
}
