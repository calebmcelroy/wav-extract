package main

import (
	"fmt"
	"github.com/calebmcelroy/wav-extract/wav"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Track struct {
	writer *wav.Writer
	file   *os.File

	Name     string
	Channels []int
}

func (t *Track) WriteAt(p []byte, off int64) (n int, err error) {
	return t.writer.WriteAt(p, off)
}

func (t *Track) Close() error {
	if err := t.writer.Close(); err != nil {
		return err
	}

	if err := t.file.Close(); err != nil {
		return err
	}

	return nil
}

func initTracks(stereoStr string, channelsStr string, outputDir string, numChans, bitDepth, sampleRate int) ([]*Track, error) {
	if stereoStr != "" && channelsStr != "" {
		return nil, fmt.Errorf("both --stereo and --channels cannot be specified, choose just one")
	}

	var tracks []*Track

	var channelPairs [][]int
	var err error

	if stereoStr != "" {
		channelPairs, err = parseChannelsString(stereoStr, numChans, false)
		if err != nil {
			return nil, err
		}
	} else if channelsStr != "" {
		channelPairs, err = parseChannelsString(channelsStr, numChans, true)
		if err != nil {
			return nil, err
		}
	}

	// Parse stereo pairs from the stereoStr
	if channelPairs != nil && len(channelPairs) > 0 {
		for _, channels := range channelPairs {
			track, err := newTrack(channels, outputDir, sampleRate, bitDepth)

			if err != nil {
				return nil, err
			}

			tracks = append(tracks, track)
		}
	}

	if channelsStr == "" {
		// Add mono tracks for any channels not included in stereo pairs

		usedChannels := make(map[int]bool)
		for _, channels := range channelPairs {
			for _, channel := range channels {
				usedChannels[channel+1] = true
			}
		}

		for ch := 1; ch <= numChans; ch++ {
			if !usedChannels[ch] {
				track, err := newTrack([]int{ch - 1}, outputDir, sampleRate, bitDepth)
				if err != nil {
					return nil, err
				}
				tracks = append(tracks, track)
			}
		}
	}

	return tracks, nil
}

func parseChannelsString(str string, numChans int, allowMono bool) ([][]int, error) {
	usedChannels := make(map[int]bool)
	var channels [][]int

	pairs := strings.Split(str, ",")
	for _, pairStr := range pairs {
		channelsStr := strings.Split(pairStr, "/")
		stereo := len(channelsStr) == 2
		mono := len(channelsStr) == 1

		if len(channelsStr) != 2 && !(allowMono && mono) {
			if allowMono {
				return nil, fmt.Errorf("invalid stereo pair format: %s", pairStr)
			}
			return nil, fmt.Errorf("invalid stereo pair format: %s", pairStr)
		}

		leftChan, err := strconv.Atoi(channelsStr[0])
		if err != nil {
			return nil, fmt.Errorf("invalid channel number: %s", channelsStr[0])
		}

		var rightChan int
		if stereo {
			rightChan, err = strconv.Atoi(channelsStr[1])
			if err != nil {
				return nil, fmt.Errorf("invalid channel number: %s", channelsStr[1])
			}
		}

		// Validate channels are within range
		if leftChan < 1 || leftChan > numChans || (stereo && (rightChan < 1 || rightChan > numChans)) {
			return nil, fmt.Errorf("channel numbers must be between 1 and %d", numChans)
		}

		// Validate left != right
		if stereo && leftChan == rightChan {
			return nil, fmt.Errorf("left channel must be less than right channel in pair: %s", pairStr)
		}

		// Check for duplicate channels
		if usedChannels[leftChan] || (stereo && usedChannels[rightChan]) {
			return nil, fmt.Errorf("duplicate channel in pair: %s", pairStr)
		}
		usedChannels[leftChan] = true
		usedChannels[rightChan] = true

		if stereo {
			channels = append(channels, []int{leftChan - 1, rightChan - 1})
		} else {
			channels = append(channels, []int{leftChan - 1})
		}
	}

	return channels, nil
}

func newTrack(channels []int, outputDir string, sampleRate, bitsPerSample int) (*Track, error) {
	var name string
	if len(channels) == 2 {
		name = fmt.Sprintf("track_%dL_%dR.wav", channels[0]+1, channels[1]+1)
	} else {
		name = fmt.Sprintf("track_%d.wav", channels[0]+1)
	}

	outFilePath := filepath.Join(outputDir, name)
	outFile, err := os.Create(outFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file '%s': %v", outFilePath, err)
	}

	wavWriter := wav.NewWriter(outFile, 1, len(channels), sampleRate, bitsPerSample)

	return &Track{
		wavWriter,
		outFile,
		name,
		channels, // Zero-based indexing
	}, nil
}
