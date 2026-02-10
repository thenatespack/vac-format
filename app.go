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
	"strings"

	"github.com/dhowden/tag"
)

const (
	magicNumber    = "CSNG"
	versionNumber  = uint32(1)
	keySizeBytes   = 32
	strFieldLength = 64
)

const headerSize = 4 + 4 + 4 + strFieldLength*3 + 8 + 4 + 4 + 4 // TOTP field removed

type Player interface {
	Play(io.Reader) error
}

type FFPlayPlayer struct{}
type VLCPlayer struct{}
type MPVPlayer struct{}

// 🔥 SINGLE PASSPHRASE - DEFAULT "hello mario"
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
		vacFile := playCmd.String("file", "", "Path to VAC file")
		playerName := playCmd.String("player", detectDefaultPlayer(), "Player: ffplay, vlc, mpv")
		playCmd.Parse(os.Args[2:])
		if *vacFile == "" {
			*vacFile = os.Args[2]
		}
		var player Player
		switch strings.ToLower(*playerName) {
		case "vlc":
			player = VLCPlayer{}
		case "mpv":
			player = MPVPlayer{}
		default:
			player = FFPlayPlayer{}
		}
		err := Play(*vacFile, player)
		if err != nil {
			log.Fatal(err)
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
	fmt.Println("CipherSong VAC CLI v2.0 - NO TOTP ✅")
	fmt.Println("Commands:")
	fmt.Println("  encode -flac <file|folder> [-output <file|folder>] [-batch]  Encode FLAC to VAC")
	fmt.Println("  play <file.vac> [-player ffplay|vlc|mpv]                       Play VAC file")
	fmt.Println("  info <file.vac>                                              Show VAC file info")
	fmt.Println("\nFlags:")
	fmt.Println("  -passphrase <pass>    Encryption passphrase (default: hello mario)")
	fmt.Println("  -batch                Process folder (FLAC files only)")
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

// BATCH ENCODE
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
	
	keyBytes := deriveKey([]byte(passphrase))
	fmt.Printf("🔍 Encoding with passphrase: '%s' (%d chars)\n", passphrase, len(passphrase))
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
		return err
	}

	_, err = out.Write(encData)
	return err
}

func createHeader(title, artist, album string, duration float64, bitrate, sampleRate, track int) []byte {
	buf := make([]byte, headerSize)
	
	copy(buf[0:4], []byte(magicNumber))
	binary.BigEndian.PutUint32(buf[4:8], versionNumber)
	binary.BigEndian.PutUint32(buf[8:12], keySizeBytes)

	offset := 12
	copy(buf[offset:offset+strFieldLength], padOrTrim(title, strFieldLength))
	offset += strFieldLength
	copy(buf[offset:offset+strFieldLength], padOrTrim(artist, strFieldLength))
	offset += strFieldLength
	copy(buf[offset:offset+strFieldLength], padOrTrim(album, strFieldLength))
	offset += strFieldLength
	binary.BigEndian.PutUint64(buf[offset:offset+8], math.Float64bits(duration))
	offset += 8
	binary.BigEndian.PutUint32(buf[offset:offset+4], uint32(bitrate))
	offset += 4
	binary.BigEndian.PutUint32(buf[offset:offset+4], uint32(sampleRate))
	offset += 4
	binary.BigEndian.PutUint32(buf[offset:offset+4], uint32(track))

	return buf
}

func padOrTrim(s string, length int) []byte {
	if len(s) > length {
		return []byte(s[:length])
	}
	padded := make([]byte, length)
	copy(padded, s)
	return padded
}

func deriveKey(pass []byte) []byte {
	h := sha512.New()
	h.Write(pass)
	return h.Sum(nil)[:keySizeBytes]
}

func Play(path string, player Player) error {
	fmt.Printf("🔍 Using passphrase: '%s' (%d chars)\n", passphrase, len(passphrase))
	keyBytes := deriveKey([]byte(passphrase))
	fmt.Printf("🔑 Derived key: %x\n", keyBytes[:16])

	// 🔥 READ ENTIRE FILE AT ONCE
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	fmt.Printf("📦 Total file: %d bytes\n", len(data))
	
	// 🔥 PARSE HEADER FROM BEGINNING OF FILE
	if len(data) < headerSize {
		return fmt.Errorf("file too short for header")
	}
	
	headerBytes := data[:headerSize]
	if string(headerBytes[:4]) != magicNumber {
		return fmt.Errorf("not a VAC file")
	}

	// Parse header fields
	offset := 12
	title := strings.TrimRight(string(headerBytes[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	artist := strings.TrimRight(string(headerBytes[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	album := strings.TrimRight(string(headerBytes[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	encData := data[headerSize:]
	fmt.Printf("🔓 Decrypting %d bytes of encrypted data (nonce=%d)\n", len(encData), 12)
	
	decrypted, err := decrypt(encData, keyBytes)
	if err != nil {
		return fmt.Errorf("audio decryption failed: %w", err)
	}

	fmt.Printf("✅ SUCCESS - Decrypted %d bytes\n", len(decrypted))
	fmt.Printf("🎵 Playing: %s - %s (%s)\n", title, artist, album)
	return player.Play(bytes.NewReader(decrypted))
}


func gcmNonceSize() int {
	block, _ := aes.NewCipher(make([]byte, 32))
	gcm, _ := cipher.NewGCM(block)
	return gcm.NonceSize()
}



type VacHeader struct {
	title, artist, album string
	duration    float64
	bitrate, sampleRate, track int
}


func readHeader(path string) (*VacHeader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	header := make([]byte, headerSize)  // 252 bytes
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, err
	}
	
	if string(header[:4]) != magicNumber {
		return nil, fmt.Errorf("not a VAC file")
	}

	// 🔥 FIXED: Start at CORRECT offset (skip magic+version+keysize = 12 bytes)
	offset := 12
	title := strings.TrimRight(string(header[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	artist := strings.TrimRight(string(header[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	album := strings.TrimRight(string(header[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	duration := math.Float64frombits(binary.BigEndian.Uint64(header[offset : offset+8]))
	offset += 8
	bitrate := int(binary.BigEndian.Uint32(header[offset : offset+4]))
	offset += 4
	sampleRate := int(binary.BigEndian.Uint32(header[offset : offset+4]))
	offset += 4
	track := int(binary.BigEndian.Uint32(header[offset : offset+4]))

	return &VacHeader{
		title:       title,
		artist:      artist,
		album:       album,
		duration:    duration,
		bitrate:     bitrate,
		sampleRate:  sampleRate,
		track:       track,
	}, nil
}



// Player implementations
func (FFPlayPlayer) Play(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	tmpfile, err := os.CreateTemp("", "vac-*.flac")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()
	
	if _, err := tmpfile.Write(data); err != nil {
		return err
	}
	
	cmd := exec.Command("ffplay", "-nodisp", "-autoexit", tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (VLCPlayer) Play(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	tmpfile, err := os.CreateTemp("", "vac-*.flac")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()
	
	if _, err := tmpfile.Write(data); err != nil {
		return err
	}
	
	cmd := exec.Command("vlc", "--play-and-exit", tmpfile.Name())
	return cmd.Run()
}

func (MPVPlayer) Play(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	tmpfile, err := os.CreateTemp("", "vac-*.flac")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()
	
	if _, err := tmpfile.Write(data); err != nil {
		return err
	}
	
	cmd := exec.Command("mpv", "--no-video", "--no-terminal", tmpfile.Name())
	return cmd.Run()
}

func info(path string) {
	fi, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}

	header, err := readHeader(path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("🎵 CipherSong VAC v%d\n", versionNumber)
	fmt.Printf("📁 File: %s (%.2f MB)\n", path, float64(fi.Size())/1024/1024)
	fmt.Printf("🎤 Title: %s\n", header.title)
	fmt.Printf("👤 Artist: %s\n", header.artist)
	fmt.Printf("💿 Album: %s\n", header.album)
	fmt.Printf("🎼 Track: %d\n", header.track)
	fmt.Printf("⏱️  Duration: %.2fs\n", header.duration)
	fmt.Printf("🔊 Bitrate: %d kbps\n", header.bitrate)
	fmt.Printf("📊 Sample Rate: %d Hz\n", header.sampleRate)
}

func encrypt(plain, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:nonceSize]
	data := ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, data, nil)
}
