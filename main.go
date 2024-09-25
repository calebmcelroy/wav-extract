package main

import (
	"flag"
	"fmt"
	"github.com/maruel/natural"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var StartTime = time.Now()

func main() {
	inputDirFlag := flag.String("in", ".", "Folder containing input WAV files")
	outputDirFlag := flag.String("out", "", "Folder where output files will be saved")
	forceFlag := flag.Bool("force", false, "Overwrite existing files in output folder")
	stereoFlag := flag.String("stereo", "", "Stereo pairs to extract (e.g. 1/2,3/4)")
	channelsFlag := flag.String("channels", "", "Channels to extract (e.g. 1/2,5)")
	flag.Parse()

	inputDir := *inputDirFlag
	outputDir := *outputDirFlag
	force := *forceFlag

	if outputDir == "" {
		fmt.Println("Error output directory not specified. Please add parameter: --out=path/to/your/folder")
		os.Exit(1)
	}

	// throw error if output directory contains wav files
	outputDirFiles, err := filepath.Glob(filepath.Join(outputDir, "*.wav"))

	if !force && err == nil && len(outputDirFiles) > 0 {
		fmt.Println("Warning! Output folder already contains wav files. Add --force parameter if you want to overwrite files.")
		os.Exit(1)
	}

	if force {
		for _, file := range outputDirFiles {
			err := os.Remove(file)
			if err != nil {
				fmt.Printf("Error removing file %s: %v\n", file, err)
				os.Exit(1)
			}
		}
	}

	var files []string
	if strings.HasSuffix(inputDir, ".wav") {
		files = []string{inputDir}
	} else {
		files, err = filepath.Glob(filepath.Join(inputDir, "*.wav"))
		if err != nil || len(files) == 0 {
			if inputDir == "." {
				fmt.Println("Error: no wav files found in the current directory. Please consider adding parameter: --in=path/to/your/wavs or run the program in a folder containing the wav files.")
				os.Exit(1)
			}
			fmt.Println("Error: no wav files found in the input directory.")
			os.Exit(1)
		}

		sort.Sort(natural.StringSlice(files))
	}

	os.MkdirAll(outputDir, os.ModePerm)

	decoders, err := initDecoders(files)

	if err != nil {
		fmt.Printf("Error %v\n", err)
		os.Exit(1)
	}

	defer func() {
		for _, decoder := range decoders {
			decoder.File.Close()
		}
	}()

	decoder := decoders[0]
	tracks, err := initTracks(*stereoFlag, *channelsFlag, outputDir, int(decoder.NumChans), int(decoder.BitDepth), int(decoder.SampleRate))

	if err != nil {
		fmt.Println("Error initializing tracks:", err)
		os.Exit(1)
	}

	defer func() {
		for _, track := range tracks {
			track.Encoder.Close()
		}
	}()

	defer func() {
		fmt.Println("\n\nDone in", time.Since(StartTime))
	}()

	extract(decoders, tracks, time.Millisecond*500, printProgress)
}

func printProgress(p Progress) {
	if p.TotalBytes == 0 {
		return
	}
	percent := (p.BytesWritten * 100) / p.TotalBytes
	activeCharPos := int((30 * percent) / 100)
	progressStr := ""
	for pos := range 30 {
		if pos < activeCharPos {
			progressStr += "="
		} else {
			progressStr += " "
		}
	}

	bytesPerSecond := (p.BytesWritten * 1000) / time.Since(StartTime).Milliseconds()

	if bytesPerSecond == 0 {
		return
	}

	timeRemaining := time.Duration((p.TotalBytes-p.BytesWritten)/bytesPerSecond) * time.Second
	timeRemaining += time.Second
	fmt.Printf("\r%d%% [%s] %d MB/s — %v remaining", percent, progressStr, bytesPerSecond/1024/1024, timeRemaining)
}
