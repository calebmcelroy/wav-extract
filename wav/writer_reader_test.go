package wav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
)

func TestWriterThenReader(t *testing.T) {
	// create wav files. each with 2 channels, 16 bits per sample, 44100 sample rate
	// with incrementing values chan values = 1,-1,2,-2

	// create first wav file
	file1, err := os.OpenFile("small1.wav", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	wav1 := NewWriter(file1, 1, 3, 44100, 16)

	// write data to wav files
	data := &bytes.Buffer{}
	for i := 0; i < 32767/2; i++ {
		// ch 1
		binary.Write(data, binary.LittleEndian, int16(i))
		// ch 2
		binary.Write(data, binary.LittleEndian, int16(i))
		// ch 3
		binary.Write(data, binary.LittleEndian, int16(i))

		// ch 1
		binary.Write(data, binary.LittleEndian, int16(-i))
		// ch 2
		binary.Write(data, binary.LittleEndian, int16(-i))
		// ch 3
		binary.Write(data, binary.LittleEndian, int16(-i))
	}

	wav1.WriteAt(data.Bytes(), 0)
	wav1.Close()
	file1.Close()

	// test reader
	file, err := os.Open("small1.wav")
	if err != nil {
		t.Fatal(err)
	}
	wav := NewReader(file)

	err = wav.ReadHeader()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(wav.AudioFormat)
	fmt.Println(wav.NumChans)
	fmt.Println(wav.SampleRate)
	fmt.Println(wav.ByteRate)
	fmt.Println(wav.BlockAlign)
	fmt.Println(wav.BitsPerSample)
	fmt.Println(wav.DataSize)

	if wav.AudioFormat != 1 {
		t.Fatal("AudioFormat is not 1")
	}

	if wav.NumChans != 3 {
		t.Fatal("NumChans is not 3")
	}

	if wav.SampleRate != 44100 {
		t.Fatal("SampleRate is not 44100")
	}

	if wav.ByteRate != wav.SampleRate*wav.NumChans*wav.BitsPerSample/8 {
		t.Fatal("ByteRate is incorrect")
	}

	if wav.BlockAlign != wav.NumChans*wav.BitsPerSample/8 {
		t.Fatal("BlockAlign is incorrect")
	}

	if wav.BitsPerSample != 16 {
		t.Fatal("BitsPerSample is not 16")
	}

	if wav.DataSize != 196596 {
		t.Fatal("DataSize is incorrect", wav.DataSize)
	}

	dataChunk := make([]byte, wav.ByteRate)

	for {
		n, err := wav.Read(dataChunk)
		if err != nil {
			t.Fatal(err)
		}

		if n == 0 || n < len(dataChunk) {
			break
		}
	}
}
