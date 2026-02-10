package main

import (
	"flag"
	"log"
	"os"
	"strings"
)


const (
	magicNumber    = "CSNG"
	versionNumber  = uint32(1)
	keySizeBytes   = 32
	strFieldLength = 64
	headerSize     = 4 + 4 + 4 + strFieldLength*3 + 8 + 4 + 4 + 4
)

var passphrase = "hello mario"
var batchMode bool

func init() {
	flag.StringVar(&passphrase, "passphrase", "hello mario", "Encryption passphrase")
	flag.BoolVar(&batchMode, "batch", false, "Batch process folder (only FLAC files)")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	flag.Parse()
	os.Args = os.Args[1:]

	switch os.Args[0] {
	case "encode":
		inputPath := ""
		outputPath := ""
		for i := 1; i < len(os.Args); i++ {
			if os.Args[i] == "-flac" && i+1 < len(os.Args) {
				inputPath = os.Args[i+1]
				i++
			} else if os.Args[i] == "-output" && i+1 < len(os.Args) {
				outputPath = os.Args[i+1]
				i++
			}
		}
		if inputPath == "" {
			log.Fatal("Please provide FLAC file/folder with -flac")
		}
		if isDir(inputPath) || batchMode {
			batchEncode(inputPath, outputPath)
		} else {
			encode(inputPath, outputPath)
		}

	case "play":
		vacFile := os.Args[1]
		playerName := detectDefaultPlayer()
		for i := 2; i < len(os.Args); i++ {
			if os.Args[i] == "-player" && i+1 < len(os.Args) {
				playerName = os.Args[i+1]
				i++
				break
			}
		}
		var player Player
		switch strings.ToLower(playerName) {
		case "vlc":
			player = VLCPlayer{}
		case "mpv":
			player = MPVPlayer{}
		default:
			player = FFPlayPlayer{}
		}
		if err := Play(vacFile, player); err != nil {
			log.Fatal(err)
		}

	case "info":
		if len(os.Args) != 2 {
			log.Fatal("Usage: vac info <file.vac>")
		}
		info(os.Args[1])

	default:
		usage()
	}
}
