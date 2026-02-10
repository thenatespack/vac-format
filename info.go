package main

import(
	"fmt"
	"os"
	"log"
)

func info(path string) {
	fi, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}

	header, err := readHeader(path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("🎵 Vitakrypt Audio Codec(VAC) v%d\n", versionNumber)
	fmt.Printf("📁 File: %s (%.2f MB)\n", path, float64(fi.Size())/1024/1024)
	fmt.Printf("🎤 Title: %s\n", header.title)
	fmt.Printf("👤 Artist: %s\n", header.artist)
	fmt.Printf("💿 Album: %s\n", header.album)
	fmt.Printf("🎼 Track: %d\n", header.track)
	fmt.Printf("⏱️  Duration: %.2fs\n", header.duration)
	fmt.Printf("🔊 Bitrate: %d kbps\n", header.bitrate)
	fmt.Printf("📊 Sample Rate: %d Hz\n", header.sampleRate)
}

