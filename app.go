package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"bufio"

	"github.com/dhowden/tag"
)

const (
	magicNumber    = "CSNG"
	versionNumber  = uint32(1)
	keySizeBytes   = 32
	strFieldLength = 64
)

const headerSize = 4 + 4 + 4 + strFieldLength*3 + 8 + 4 + 4 + 4

type Player interface {
	Play(io.Reader) error
}

type FFPlayPlayer struct{}
type VLCPlayer struct{}
type MPVPlayer struct{}

// SINGLE PASSPHRASE - DEFAULT "hello mario"
var passphrase = "hello mario"
var batchMode bool

func init() {
	flag.StringVar(&passphrase, "passphrase", "hello mario", "Encryption passphrase")
	flag.BoolVar(&batchMode, "batch", false, "Batch process folder (only FLAC files)")
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "encode":
		encodeCmd := flag.NewFlagSet("encode", flag.ExitOnError)
		inputPath := encodeCmd.String("flac", "", "Path to FLAC file or folder")
		outputPath := encodeCmd.String("output", "", "Output VAC file or folder")
		encodeCmd.Parse(os.Args[2:])

		if *inputPath == "" {
			log.Fatal("Please provide FLAC file/folder with -flac")
		}

		if isDir(*inputPath) || batchMode {
			batchEncode(*inputPath, *outputPath)
		} else {
			encode(*inputPath, *outputPath)
		}

	case "play":
		playCmd := flag.NewFlagSet("play", flag.ExitOnError)
		inputPath := playCmd.String("file", "", "VAC file or folder")
		playerName := playCmd.String("player", detectDefaultPlayer(), "Player: ffplay, vlc, mpv")
		shuffleFlag := playCmd.Bool("shuffle", false, "Shuffle playlist")
		playCmd.Parse(os.Args[2:])

		var player Player
		switch strings.ToLower(*playerName) {
		case "vlc":
			player = VLCPlayer{}
		case "mpv":
			player = MPVPlayer{}
		default:
			player = FFPlayPlayer{}
		}

		if *inputPath == "" {
			interactivePlay(player)
		} else if isDir(*inputPath) {
			playlist := scanVacFolder(*inputPath, *shuffleFlag)
			playPlaylist(playlist, player)
		} else {
			err := Play(*inputPath, player)
			if err != nil {
				log.Fatal(err)
			}
		}

	case "info":
		if len(os.Args) != 3 {
			log.Fatal("Usage: vac info <file.vac>")
		}
		info(os.Args[2])

	default:
		usage()
	}
}

func usage() {
	fmt.Println("CipherSong VAC CLI v2.2 - AUTO-PLAY 🎧")
	fmt.Println("Commands:")
	fmt.Println("  encode -flac <file|folder> [-output <folder>] [-batch]          Encode FLAC to VAC")
	fmt.Println("  play [<folder>|file.vac] [-player ffplay|vlc|mpv] [-shuffle]    Auto-play folder/single")
	fmt.Println("  info <file.vac>                                                 Show VAC file info")
	fmt.Println("\n🚀 Auto-play examples:")
	fmt.Println("  ./vac play ~/vac_songs/                    # Play ALL 322 songs!")
	fmt.Println("  ./vac play ~/vac_songs/ -shuffle          # Shuffle 322 songs")
	fmt.Println("  ./vac play song.vac                       # Single file")
	os.Exit(1)
}

func detectDefaultPlayer() string {
	switch {
	case hasCommand("mpv"):
		return "mpv"
	case hasCommand("vlc"):
		return "vlc"
	default:
		return "ffplay"
	}
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

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
		log.Fatal("No FLAC files found!")
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

func encodeFile(flacPath, vacPath string) error {
	keyBytes := deriveKey([]byte(passphrase))
	title, artist, album, duration, bitrate, sampleRate, track, _ := readFlacMetadata(flacPath)
	return createVacFile(flacPath, vacPath, keyBytes, title, artist, album, duration, bitrate, sampleRate, track)
}

func encode(flacPath, outPath string) {
	if outPath == "" {
		outPath = strings.TrimSuffix(flacPath, ".flac") + ".vac"
	}
	
	fmt.Printf("🔍 Encoding with passphrase: '%s' (%d chars)\n", passphrase, len(passphrase))
	keyBytes := deriveKey([]byte(passphrase))
	fmt.Printf("🔑 Derived key: %x\n", keyBytes[:16])
	
	fmt.Printf("Encoding %s → %s\n", flacPath, outPath)
	if err := encodeFile(flacPath, outPath); err != nil {
		log.Fatalf("Failed to encode VAC: %v", err)
	}

	fmt.Printf("\n✅ Created: %s\n", outPath)
	fmt.Printf("🔑 Key (Base64): %s\n", base64.StdEncoding.EncodeToString(keyBytes))
}

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

func createVacFile(flacPath, outPath string, key []byte, title, artist, album string, duration float64, bitrate, sampleRate, track int) error {
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	header := createHeader(title, artist, album, duration, bitrate, sampleRate, track)
	if _, err := out.Write(header); err != nil {
		return err
	}

	data, err := os.ReadFile(flacPath)
	if err != nil {
		return err
	}

	encData, err := encrypt(data, key)
	if err != nil {
		return
