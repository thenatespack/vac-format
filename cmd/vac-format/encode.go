package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"encoding/base64"
)

func batchEncode(inputDir, outputDir string) {
	fmt.Printf("🎵 BATCH MODE: Processing folder %s\n", inputDir)
	
	if outputDir == "" {
		outputDir = inputDir
	}
	
	if !isDir(outputDir) {
		os.MkdirAll(outputDir, 0755)
	}
	
	var flacFiles []string
	filepath.WalkDir(inputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".flac") {
			flacFiles = append(flacFiles, path)
		}
		return nil
	})
	
	if len(flacFiles) == 0 {
		fmt.Println("❌ No FLAC files found!")
		os.Exit(1)
	}
	
	fmt.Printf("📂 Found %d FLAC files\n", len(flacFiles))
	
	successCount := 0
	for i, flacPath := range flacFiles {
		fmt.Printf("\n[%d/%d] 🟡 Processing %s ", i+1, len(flacFiles), filepath.Base(flacPath))
		
		vacPath := filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(flacPath), ".flac")+".vac")
		
		if err := encodeFile(flacPath, vacPath); err != nil {
			fmt.Printf("❌ FAILED\n  %v\n", err)
			continue
		}
		
		fmt.Printf("✅ DONE\n")
		successCount++
	}
	
	fmt.Printf("\n🎉 BATCH COMPLETE: %d/%d files processed successfully!\n", successCount, len(flacFiles))
}

func encode(flacPath, outPath string) {
	if outPath == "" {
		outPath = strings.TrimSuffix(flacPath, ".flac") + ".vac"
	}
	
	keyBytes := deriveKey([]byte(passphrase))
	fmt.Printf("🔍 Encoding with passphrase: '%s' (%d chars)\n", passphrase, len(passphrase))
	fmt.Printf("🔑 Derived key: %x\n", keyBytes[:16])
	
	fmt.Printf("Encoding %s → %s\n", flacPath, outPath)
	if err := encodeFile(flacPath, outPath); err != nil {
		fmt.Printf("❌ Failed to encode VAC: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Created: %s\n", outPath)
	fmt.Printf("🔑 Key (Base64): %s\n", base64.StdEncoding.EncodeToString(keyBytes))
}

func encodeFile(flacPath, vacPath string) error {
	keyBytes := deriveKey([]byte(passphrase))
	title, artist, album, duration, bitrate, sampleRate, track, _ := readFlacMetadata(flacPath)
	return createVacFile(flacPath, vacPath, keyBytes, title, artist, album, duration, bitrate, sampleRate, track)
}
