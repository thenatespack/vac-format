package main

import (
	"os"
	"github.com/dhowden/tag"
)

func readFlacMetadata(path string) (title, artist, album string, duration float64, bitrate, sampleRate, track, totalSamples int) {
	f, err := os.Open(path)
	if err != nil {
		return "Unknown", "Unknown", "Unknown", 0, 0, 0, 1, 0
	}
	defer f.Close()

	meta, err := tag.ReadFrom(f)
	if err == nil {
		if t := meta.Title(); t != "" {
			title = truncateField(t, strFieldLength)
		}
		if a := meta.Artist(); a != "" {
			artist = truncateField(a, strFieldLength)
		}
		if al := meta.Album(); al != "" {
			album = truncateField(al, strFieldLength)
		}
		track = 1
	}

	return title, artist, album, 180.0, 1411, 44100, track, 0
}

func truncateField(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen-3]) + "..."
	}
	return s
}
