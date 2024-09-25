package main

import (
	"fmt"
	"github.com/calebmcelroy/wav"
	"github.com/go-audio/audio"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type Decoder struct {
	wav.Decoder
	File *os.File
	Name string
}

type TrackWriteTask struct {
	TrackIndex   int
	PCMBuffer    *audio.IntBuffer
	BytesWritten int
	Wg           *sync.WaitGroup
}

type Progress struct {
	TotalBytes   int64
	BytesWritten int64
}

func initDecoders(files []string) ([]*Decoder, error) {
	decoders := make([]*Decoder, len(files))

	for i, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %v", file, err)
		}

		decoder := wav.NewDecoder(f)
		if decoder == nil {
			return nil, fmt.Errorf("failed to create decoder for file %s", file)
		}

		if !decoder.IsValidFile() {
			return nil, fmt.Errorf("invalid WAV file: %s", file)
		}

		decoders[i] = &Decoder{
			Decoder: *decoder,
			File:    f,
			Name:    filepath.Base(file),
		}
	}

	// make sure all channels, sample rates, & bit rates are the same
	for i := 1; i < len(decoders); i++ {
		if decoders[i].SampleRate != decoders[i-1].SampleRate {
			return nil, fmt.Errorf("sample rate mismatch: %d (%s) != %d (%s)", decoders[i].SampleRate, filepath.Base(files[i]), decoders[i-1].SampleRate, filepath.Base(files[i-1]))
		}
		if decoders[i].NumChans != decoders[i-1].NumChans {
			return nil, fmt.Errorf("number of channels: %d (%s) != %d (%s)", decoders[i].NumChans, filepath.Base(files[i]), decoders[i-1].NumChans, filepath.Base(files[i-1]))
		}
		if decoders[i].BitDepth != decoders[i-1].BitDepth {
			return nil, fmt.Errorf("bit depth mismatch: %d (%s) != %d (%s)", decoders[i].BitDepth, filepath.Base(files[i]), decoders[i-1].BitDepth, filepath.Base(files[i-1]))
		}
	}

	return decoders, nil
}

func extract(decoders []*Decoder, tracks []*Track, progressInterval time.Duration, progressFunc func(p Progress)) {
	totalBytes := int64(0)
	bytesWritten := &atomic.Int64{}
	decoderPositions := make([]int64, len(decoders))
	for i, decoder := range decoders {
		decoderPositions[i] = totalBytes / int64(decoder.NumChans)
		err := decoder.FwdToPCM()
		if err != nil {
			fmt.Printf("Error reading PCM data length: %v\n", err)
			os.Exit(1)
		}
		totalBytes += decoder.PCMLen()
	}

	totalTrackChannels := int64(0)
	for _, track := range tracks {
		totalTrackChannels += int64(len(track.Channels))
	}

	totalBytes = (totalBytes * totalTrackChannels) / int64(decoders[0].NumChans)

	done := false
	wg := sync.WaitGroup{}
	wg.Add(len(decoders))

	// report progress
	go func() {
		for {
			time.Sleep(progressInterval)

			if done {
				break
			}

			progressFunc(Progress{
				TotalBytes:   totalBytes,
				BytesWritten: bytesWritten.Load(),
			})
		}
	}()

	// decode in parallel
	for i, decoder := range decoders {
		go func() {
			defer wg.Done()
			err := extractTracks(decoder, tracks, bytesWritten, decoderPositions[i])
			if err != nil {
				fmt.Println()
				fmt.Printf("Error processing file %s: %v\n", decoder.Name, err)
				os.Exit(1)
			}
		}()
	}

	wg.Wait()
	done = true
}

func extractTracks(decoder *Decoder, tracks []*Track, bytesWritten *atomic.Int64, tracksPos int64) error {
	numChannels := int(decoder.NumChans)
	sampleRate := int(decoder.SampleRate)
	bitDepth := int(decoder.BitDepth)

	bytesPerSecond := sampleRate * (bitDepth / 8)
	chunkSize := bytesPerSecond

	intBufferPool := sync.Pool{
		New: func() interface{} {
			return &audio.IntBuffer{
				Data: make([]int, chunkSize*numChannels),
				Format: &audio.Format{
					SampleRate:  sampleRate,
					NumChannels: numChannels,
				},
				SourceBitDepth: bitDepth,
			}
		},
	}

	trackPos := make([]int64, len(tracks))
	for i, track := range tracks {
		trackPos[i] = tracksPos * int64(len(track.Channels))
	}

	trackBuffers := make([]*audio.IntBuffer, len(tracks))
	for i, track := range tracks {
		trackBuffers[i] = &audio.IntBuffer{
			Data: make([]int, chunkSize*len(track.Channels)),
			Format: &audio.Format{
				SampleRate:  sampleRate,
				NumChannels: len(track.Channels),
			},
			SourceBitDepth: bitDepth,
		}
	}

	trackChans := make([]chan TrackWriteTask, len(tracks))
	for i := range tracks {
		trackChans[i] = make(chan TrackWriteTask, len(tracks))
	}

	for trackIndex, track := range tracks {
		go func() {
			for task := range trackChans[trackIndex] {
				for i := 0; i < task.BytesWritten; i += numChannels {
					for j, ch := range track.Channels {
						trackDataIndex := (i/numChannels)*len(track.Channels) + j
						trackBuffers[trackIndex].Data[trackDataIndex] = task.PCMBuffer.Data[i+ch]
					}
				}

				n, err := track.Encoder.WriteAt(trackBuffers[trackIndex], trackPos[trackIndex])
				if err != nil {
					fmt.Printf("Failed to write %s: %v\n", track.Name, err)
					os.Exit(1)
				}

				trackPos[trackIndex] += n
				bytesWritten.Add(n)
				task.Wg.Done()
			}
		}()
	}

	for {
		if decoder.EOF() {
			break
		}

		rawPCMBuffer := intBufferPool.Get().(*audio.IntBuffer)
		defer intBufferPool.Put(rawPCMBuffer)
		n, err := decoder.PCMBuffer(rawPCMBuffer)

		if err != nil {
			return fmt.Errorf("failed to read PCM data: %v", err)
		}

		if n == 0 {
			break
		}

		wg := &sync.WaitGroup{}
		wg.Add(len(tracks))
		for i := range tracks {
			trackChans[i] <- TrackWriteTask{
				TrackIndex:   i,
				PCMBuffer:    rawPCMBuffer,
				Wg:           wg,
				BytesWritten: n,
			}
		}

		wg.Wait()
	}

	for i := range tracks {
		close(trackChans[i])
	}

	return nil
}
