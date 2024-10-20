package main

import (
	"context"
	"fmt"
	"github.com/calebmcelroy/wav-extract/wav"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type WavFile struct {
	file *os.File
	*wav.Reader
	Name string
}

func (w *WavFile) Close() error {
	return w.file.Close()
}

type TrackWriteTask struct {
	TrackIndex   int
	Buffer       []byte
	BytesWritten int
	Wg           *sync.WaitGroup
}

type Progress struct {
	TotalBytes   int64
	CurrentBytes int64
}

func initReaders(files []string) ([]*WavFile, error) {
	wavFiles := make([]*WavFile, len(files))

	for i, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %v", file, err)
		}

		wavReader := wav.NewReader(f)
		wavFile := WavFile{f, wavReader, filepath.Base(file)}

		if err = wavFile.ReadHeader(); err != nil {
			return nil, fmt.Errorf("invalid WAV file (%s): %w", file, err)
		}

		wavFiles[i] = &wavFile
	}

	// make sure all channels, sample rates, & bit rates are the same
	for i := 1; i < len(wavFiles); i++ {
		if wavFiles[i].SampleRate != wavFiles[i-1].SampleRate {
			return nil, fmt.Errorf("sample rate mismatch: %d (%s) != %d (%s)", wavFiles[i].SampleRate, filepath.Base(files[i]), wavFiles[i-1].SampleRate, filepath.Base(files[i-1]))
		}
		if wavFiles[i].NumChans != wavFiles[i-1].NumChans {
			return nil, fmt.Errorf("number of channels: %d (%s) != %d (%s)", wavFiles[i].NumChans, filepath.Base(files[i]), wavFiles[i-1].NumChans, filepath.Base(files[i-1]))
		}
		if wavFiles[i].BitsPerSample != wavFiles[i-1].BitsPerSample {
			return nil, fmt.Errorf("bit depth mismatch: %d (%s) != %d (%s)", wavFiles[i].BitsPerSample, filepath.Base(files[i]), wavFiles[i-1].BitsPerSample, filepath.Base(files[i-1]))
		}
	}

	return wavFiles, nil
}

func extract(ctx context.Context, wavFiles []*WavFile, tracks []*Track, progressInterval time.Duration, progressFunc func(p Progress)) {
	totalBytes := int64(0)
	bytesProcessed := &atomic.Int64{}
	wavFilePositions := make([]int64, len(wavFiles))
	for i, wavFile := range wavFiles {
		wavFilePositions[i] = totalBytes / int64(wavFile.NumChans)
		err := wavFile.ReadHeader()
		if err != nil {
			fmt.Printf("Error reading PCM data length: %v\n", err)
			os.Exit(1)
		}
		totalBytes += int64(wavFile.DataSize)
	}

	done := false
	wg := sync.WaitGroup{}
	wg.Add(len(wavFiles))

	//report progress
	go func() {
		for {
			time.Sleep(progressInterval)

			if done {
				break
			}

			progressFunc(Progress{
				TotalBytes:   totalBytes,
				CurrentBytes: bytesProcessed.Load(),
			})
		}
	}()

	intBufferPool := &sync.Pool{
		New: func() interface{} {
			return make([]byte, wavFiles[0].ByteRate)
		},
	}

	for i, wavFile := range wavFiles {
		go func() {
			defer wg.Done()
			err := extractTracks(ctx, wavFile, tracks, intBufferPool, bytesProcessed, wavFilePositions[i])
			if err != nil {
				fmt.Println()
				fmt.Printf("Error processing file %s: %v\n", wavFile.Name, err)
				os.Exit(1)
			}
		}()
	}

	wg.Wait()
	done = true
}

func extractTracks(ctx context.Context, wavFile *WavFile, tracks []*Track, bufPool *sync.Pool, bytesProcessed *atomic.Int64, tracksPos int64) error {
	trackPos := make([]int64, len(tracks))
	for i, track := range tracks {
		trackPos[i] = tracksPos * int64(len(track.Channels))
	}

	trackBuffers := make([][]byte, len(tracks))
	for i := range tracks {
		trackBuffers[i] = make([]byte, wavFile.ByteRate)
	}

	trackChans := make([]chan TrackWriteTask, len(tracks))
	for i := range tracks {
		trackChans[i] = make(chan TrackWriteTask, 1)
	}

	for trackIndex, track := range tracks {
		go func() {
			for task := range trackChans[trackIndex] {
				if ctx.Err() != nil {
					os.Exit(1)
				}

				bytesPerSample := wavFile.BitsPerSample / 8
				trackBlockAlign := bytesPerSample * len(track.Channels)
				bufSize := 0
				for i := 0; i < task.BytesWritten; i += wavFile.BlockAlign {
					for j, channelIndex := range track.Channels {
						channelOffset := i + channelIndex*bytesPerSample
						trackOffset := (i/wavFile.BlockAlign)*trackBlockAlign + j*bytesPerSample
						bufSize += copy(trackBuffers[trackIndex][trackOffset:], task.Buffer[channelOffset:channelOffset+bytesPerSample])
					}
				}

				n, err := track.WriteAt(trackBuffers[trackIndex][:bufSize], trackPos[trackIndex])
				if err != nil {
					fmt.Printf("Failed to write %s: %v\n", track.Name, err)
					os.Exit(1)
				}

				trackPos[trackIndex] += int64(n)
				task.Wg.Done()
			}
		}()
	}

	for {
		if ctx.Err() != nil {
			os.Exit(1)
		}

		buffer := bufPool.Get().([]byte)
		defer bufPool.Put(buffer)
		n, err := wavFile.Read(buffer)

		if err == io.EOF {
			break
		}

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
				Buffer:       buffer,
				Wg:           wg,
				BytesWritten: n,
			}
		}
		wg.Wait()

		bytesProcessed.Add(int64(n))
		bufPool.Put(buffer)
	}

	for i := range tracks {
		close(trackChans[i])
	}

	return nil
}
