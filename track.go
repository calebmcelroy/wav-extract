package main

import (
	"fmt"
	"github.com/calebmcelroy/wav"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Track struct {
	Name     string
	Channels []int
	Encoder  *wav.Encoder
}

func initTracks(stereoStr string, outputDir string, numChans, bitDepth, sampleRate int) ([]*Track, error) {
	usedChannels := make(map[int]bool)
	var tracks []*Track

	// Parse stereo pairs from the stereoStr
	if stereoStr != "" {
		pairs := strings.Split(stereoStr, ",")
		for _, pairStr := range pairs {
			channelsStr := strings.Split(pairStr, "/")
			if len(channelsStr) != 2 {
				return nil, fmt.Errorf("invalid stereo pair format: %s", pairStr)
			}

			leftChan, err := strconv.Atoi(channelsStr[0])
			if err != nil {
				return nil, fmt.Errorf("invalid channel number: %s", channelsStr[0])
			}

			rightChan, err := strconv.Atoi(channelsStr[1])
			if err != nil {
				return nil, fmt.Errorf("invalid channel number: %s", channelsStr[1])
			}

			// Validate channels are within range
			if leftChan < 1 || leftChan > numChans || rightChan < 1 || rightChan > numChans {
				return nil, fmt.Errorf("channel numbers must be between 1 and %d", numChans)
			}

			// Validate left < right
			if leftChan == rightChan {
				return nil, fmt.Errorf("left channel must be less than right channel in pair: %s", pairStr)
			}

			// Check for duplicate channels
			if usedChannels[leftChan] || usedChannels[rightChan] {
				return nil, fmt.Errorf("duplicate channel in pair: %s", pairStr)
			}
			usedChannels[leftChan] = true
			usedChannels[rightChan] = true

			// Create Track for the stereo pair
			track := &Track{
				Name:     fmt.Sprintf("track_%dL_%dR.wav", leftChan, rightChan),
				Channels: []int{leftChan - 1, rightChan - 1}, // Zero-based indexing
			}

			err = initTrackEncoder(track, outputDir, sampleRate, bitDepth)
			if err != nil {
				return nil, err
			}

			tracks = append(tracks, track)
		}
	}

	// Add mono tracks for any channels not included in stereo pairs
	for ch := 1; ch <= numChans; ch++ {
		if !usedChannels[ch] {
			track := &Track{
				Name:     fmt.Sprintf("track_%d.wav", ch),
				Channels: []int{ch - 1}, // Zero-based indexing
			}
			err := initTrackEncoder(track, outputDir, sampleRate, bitDepth)
			if err != nil {
				return nil, err
			}
			tracks = append(tracks, track)
		}
	}

	return tracks, nil
}

func initTrackEncoder(t *Track, outputDir string, sampleRate, bitDepth int) error {
	outFilePath := filepath.Join(outputDir, t.Name)
	outFile, err := os.Create(outFilePath)
	if err != nil {
		return fmt.Errorf("failed to create output file '%s': %v", outFilePath, err)
	}

	// Initialize the encoder with the appropriate number of channels
	t.Encoder = wav.NewEncoder(outFile, sampleRate, bitDepth, len(t.Channels), 1)

	return nil
}
