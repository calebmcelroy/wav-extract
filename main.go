package main

import (
	"flag"
	"fmt"
	"github.com/calebmcelroy/wav"
	"github.com/go-audio/audio"
	"github.com/maruel/natural"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	start := time.Now()

	inputDirFlag := flag.String("in", ".", "Folder containing input WAV files")
	outputDirFlag := flag.String("out", "", "Folder where output files will be saved")
	flag.Parse()

	inputDir := *inputDirFlag
	outputDir := *outputDirFlag

	if outputDir == "" {
		fmt.Println("Error output directory not specified. Please add parameter: --out=path/to/your/folder")
		os.Exit(1)
	}

	files, err := filepath.Glob(filepath.Join(inputDir, "*.wav"))
	if err != nil || len(files) == 0 {
		if inputDir == "." {
			fmt.Println("Error: no wav files found in the current directory. Please consider adding parameter: --in=path/to/your/wavs or run the program in a folder containing the wav files.")
			os.Exit(1)
		}
		fmt.Println("Error: no wav files found in the input directory.")
		os.Exit(1)
	}

	sort.Sort(natural.StringSlice(files))

	os.MkdirAll(outputDir, os.ModePerm)

	decoders, resources, err := decodersFromFiles(files)

	if err != nil {
		fmt.Printf("Error %v\n", err)
		os.Exit(1)
	}

	defer func() {
		for _, resource := range resources {
			resource.Close()
		}
	}()

	tracks, err := trackEncoders(decoders, outputDir)

	if err != nil {
		fmt.Println("Error creating track encoders:", err)
		os.Exit(1)
	}

	defer func() {
		for _, trackEncoder := range tracks {
			trackEncoder.Close()
		}
	}()

	defer func() {
		fmt.Println("\n\nDone in", time.Since(start))
	}()

	totalBytes := int64(0)
	bytesWritten := &atomic.Int64{}
	decoderPositions := make([]int64, len(decoders))
	for i, decoder := range decoders {
		decoderPositions[i] = totalBytes / int64(decoder.NumChans)
		decoder.FwdToPCM()
		totalBytes += decoder.PCMLen()
	}

	done := false
	wg := sync.WaitGroup{}
	wg.Add(len(decoders))

	go func() {
		for {
			time.Sleep(time.Millisecond * 500)
			if done {
				break
			}
			percent := (bytesWritten.Load() * 100) / totalBytes
			activeCharPos := int((30 * percent) / 100)
			progressStr := ""
			for pos := range 30 {
				if pos < activeCharPos {
					progressStr += "="
				} else {
					progressStr += " "
				}
			}

			bytesPerSecond := (bytesWritten.Load() * 1000) / time.Since(start).Milliseconds()
			timeRemaining := time.Duration((totalBytes-bytesWritten.Load())/bytesPerSecond) * time.Second
			timeRemaining += time.Second
			fmt.Printf("\r%d%% [%s] %d MB/s — %v remaining", percent, progressStr, bytesPerSecond/1024/1024, timeRemaining)
		}
	}()

	for i, decoder := range decoders {
		go func() {
			defer wg.Done()
			err := extractTracks(files[i], decoder, tracks, bytesWritten, decoderPositions[i])
			if err != nil {
				fmt.Println()
				fmt.Printf("Error processing file %s: %v\n", files[i], err)
				os.Exit(1)
			}
		}()
	}

	wg.Wait()
	done = true
}

func decodersFromFiles(files []string) ([]*wav.Decoder, []*os.File, error) {
	decoders := make([]*wav.Decoder, len(files))
	resources := make([]*os.File, len(files))

	for i, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open file %s: %v", file, err)
		}

		decoder := wav.NewDecoder(f)
		if decoder == nil {
			return nil, nil, fmt.Errorf("failed to create decoder for file %s", file)
		}

		if !decoder.IsValidFile() {
			return nil, nil, fmt.Errorf("invalid WAV file: %s", file)
		}

		decoders[i] = decoder
		resources[i] = f
	}

	// make sure all channels, sample rates, & bit rates are the same
	for i := 1; i < len(decoders); i++ {
		if decoders[i].SampleRate != decoders[i-1].SampleRate {
			return nil, nil, fmt.Errorf("sample rate mismatch: %d (%s) != %d (%s)", decoders[i].SampleRate, filepath.Base(files[i]), decoders[i-1].SampleRate, filepath.Base(files[i-1]))
		}
		if decoders[i].NumChans != decoders[i-1].NumChans {
			return nil, nil, fmt.Errorf("number of channels: %d (%s) != %d (%s)", decoders[i].NumChans, filepath.Base(files[i]), decoders[i-1].NumChans, filepath.Base(files[i-1]))
		}
		if decoders[i].BitDepth != decoders[i-1].BitDepth {
			return nil, nil, fmt.Errorf("bit depth mismatch: %d (%s) != %d (%s)", decoders[i].BitDepth, filepath.Base(files[i]), decoders[i-1].BitDepth, filepath.Base(files[i-1]))
		}
	}

	return decoders, resources, nil
}

func trackEncoders(decoders []*wav.Decoder, outputDir string) ([]*wav.Encoder, error) {
	decoder := decoders[0]
	numChannels := int(decoder.NumChans)
	sampleRate := int(decoder.SampleRate)
	bitDepth := int(decoder.BitDepth)

	tracks := make([]*wav.Encoder, numChannels)

	for i := 0; i < numChannels; i++ {
		outFilePath := filepath.Join(outputDir, fmt.Sprintf("track_%d.wav", i+1))
		outFile, err := os.Create(outFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %v", err)
		}
		tracks[i] = wav.NewEncoder(outFile, sampleRate, bitDepth, 1, 1)
	}

	return tracks, nil
}

type TrackWriteTask struct {
	TrackIndex   int
	PCMBuffer    *audio.IntBuffer
	BytesWritten int
	Wg           *sync.WaitGroup
}

func extractTracks(filePath string, decoder *wav.Decoder, tracks []*wav.Encoder, bytesWritten *atomic.Int64, tracksPos int64) error {
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

	trackPos := make([]int64, numChannels)
	for i := range numChannels {
		trackPos[i] = tracksPos
	}

	trackBuffers := make([]*audio.IntBuffer, numChannels)
	for ch := range numChannels {
		trackBuffers[ch] = &audio.IntBuffer{
			Data:           make([]int, chunkSize),
			Format:         &audio.Format{SampleRate: sampleRate, NumChannels: 1},
			SourceBitDepth: bitDepth,
		}
	}

	// start goroutines to write to each track
	trackChans := make([]chan TrackWriteTask, numChannels)
	for i := range numChannels {
		trackChans[i] = make(chan TrackWriteTask, numChannels)
	}

	for track := range numChannels {
		go func() {
			for task := range trackChans[track] {
				for i := track; i < task.BytesWritten; i += numChannels {
					trackBuffers[track].Data[i/numChannels] = task.PCMBuffer.Data[i]
				}

				n, err := tracks[track].WriteAt(trackBuffers[track], trackPos[track])
				if err != nil {
					fmt.Printf("Failed to write to track %d: %v\n", track+1, err)
					os.Exit(1)
				}
				trackPos[track] += n
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
		n, err := decoder.PCMBuffer(rawPCMBuffer)

		if err != nil {
			intBufferPool.Put(rawPCMBuffer)
			return fmt.Errorf("failed to read PCM data: %v", err)
		}

		if n == 0 {
			break
		}

		wg := &sync.WaitGroup{}
		wg.Add(numChannels)
		for track := range numChannels {
			trackChans[track] <- TrackWriteTask{
				TrackIndex:   track,
				PCMBuffer:    rawPCMBuffer,
				Wg:           wg,
				BytesWritten: n,
			}
		}

		wg.Wait()
		intBufferPool.Put(rawPCMBuffer)
	}

	for track := range numChannels {
		close(trackChans[track])
	}

	return nil
}
