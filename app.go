package main

import (
	//"bytes"
	crand "crypto/rand"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	mrand "math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/jacopo-degattis/flacgo"
)

const (
	magicNumber      = "CSNG"
	versionNumber    = uint32(0)
	keySizeBytes     = 32
	totpSecretLength = 64
	strFieldLength   = 64
)

var passphrase = "hello mario"

const headerSize = 4 + 4 + 4 + totpSecretLength + strFieldLength*3 + 8 + 4 + 4 + 4

func main() {
	mrand.Seed(0)

	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "encode":
		encodeCmd := flag.NewFlagSet("encode", flag.ExitOnError)
		flacFile := encodeCmd.String("flac", "", "Path to the FLAC file")
		outputFile := encodeCmd.String("output", "CipherSong.vac", "Output VAC file")
		encodeCmd.Parse(os.Args[2:])
		if *flacFile == "" {
			log.Fatal("Please provide a FLAC file with -flac")
		}
		encode(*flacFile, *outputFile)

	case "play":
		if len(os.Args) != 3 {
			log.Fatal("Usage: vac play <file.vac>")
		}
		play(os.Args[2])

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
	fmt.Println("VAC CLI")
	fmt.Println("Commands:")
	fmt.Println("  encode -flac <file> [-output <file>]  Encode a FLAC file into VAC format")
	fmt.Println("  play <file.vac>                        Play a VAC file using afplay")
	fmt.Println("  info <file.vac>                        Show info about a VAC file")
	os.Exit(1)
}

func encode(flacPath, outPath string) {
	totpSecret := randomString(totpSecretLength)
	keyBytes := sha256.Sum256([]byte(passphrase))
	encKeyBase64 := base64.StdEncoding.EncodeToString(keyBytes[:])

	title, artist, album, duration, bitrate, sampleRate, track := readMetadata(flacPath)

	if err := createVacFile(flacPath, outPath, totpSecret, keyBytes[:], title, artist, album, duration, bitrate, sampleRate, track); err != nil {
		log.Fatalf("Failed to encode VAC: %v", err)
	}

	fmt.Printf("VAC file created: %s\n", outPath)
	fmt.Printf("Encryption Key (Base64): %s\n", encKeyBase64)
	fmt.Printf("Metadata - Title: %s, Artist: %s, Album: %s, Duration: %.2fs, Bitrate: %d, SampleRate: %d, Track: %d\n",
		title, artist, album, duration, bitrate, sampleRate, track)
}

func readMetadata(path string) (title, artist, album string, duration float64, bitrate, sampleRate, track int) {
	title, artist, album = "", "", ""
	duration, bitrate, sampleRate, track = 0, 0, 0, 0

	// Try Vorbis comments first
	if f, err := flacgo.Open(path); err == nil {
		if t, err := f.ReadMetadata("TITLE"); err == nil && t != nil {
			title = *t
		}
		if a, err := f.ReadMetadata("ARTIST"); err == nil && a != nil {
			artist = *a
		}
		if al, err := f.ReadMetadata("ALBUM"); err == nil && al != nil {
			album = *al
		}
	}

	// macOS metadata via mdls
	out, err := exec.Command("mdls", "-name", "kMDItemTitle", "-name", "kMDItemAuthors", "-name", "kMDItemAlbum", "-name", "kMDItemDurationSeconds", "-name", "kMDItemAudioBitRate", "-name", "kMDItemAudioSampleRate", "-name", "kMDItemAudioTrackNumber", path).Output()
	if err == nil {
		mdTitle := extractTitle(string(out))
		mdArtist := extractArtist(string(out))
		mdAlbum := extractAlbum(string(out))
		mdDur := extractDuration(string(out))
		mdBR := extractBitrate(string(out))
		mdSR := extractSampleRate(string(out))
		mdTrack := extractTrack(string(out))

		title = fallbackString(title, mdTitle)
		artist = fallbackString(artist, mdArtist)
		album = fallbackString(album, mdAlbum)
		duration = fallbackFloat(duration, parseFloat(mdDur))
		bitrate = fallbackInt(bitrate, parseInt(mdBR))
		sampleRate = fallbackInt(sampleRate, parseInt(mdSR))
		track = fallbackInt(track, parseInt(mdTrack))
	}

	if strings.TrimSpace(title) == "" {
		title = "Unknown"
	}
	if strings.TrimSpace(artist) == "" {
		artist = "Unknown"
	}
	if strings.TrimSpace(album) == "" {
		album = "Unknown"
	}
	if track == 0 {
		track = 1
	}

	return
}

func extractTitle(fullOutput string) string {
	return extractQuoted(fullOutput, "kMDItemTitle")
}

func extractArtist(fullOutput string) string {
	return extractQuoted(fullOutput, "kMDItemAuthors")
}

func extractAlbum(fullOutput string) string {
	return extractQuoted(fullOutput, "kMDItemAlbum")
}

func extractDuration(fullOutput string) string {
	return extractNumber(fullOutput, "kMDItemDurationSeconds")
}

func extractBitrate(fullOutput string) string {
	return extractNumber(fullOutput, "kMDItemAudioBitRate")
}

func extractSampleRate(fullOutput string) string {
	return extractNumber(fullOutput, "kMDItemAudioSampleRate")
}

func extractTrack(fullOutput string) string {
	return extractNumber(fullOutput, "kMDItemAudioTrackNumber")
}

func extractQuoted(fullOutput, key string) string {
	idx := strings.Index(fullOutput, key)
	if idx == -1 {
		return ""
	}
	value := fullOutput[idx:]
	start := strings.Index(value, "\"")
	if start == -1 {
		return ""
	}
	end := strings.Index(value[start+1:], "\"") + start + 1
	if end > start+1 {
		return value[start+1 : end]
	}
	return ""
}

func extractNumber(fullOutput, key string) string {
	idx := strings.Index(fullOutput, key)
	if idx == -1 {
		return ""
	}
	eqIdx := strings.Index(fullOutput[idx:], "=")
	if eqIdx == -1 {
		return ""
	}
	eqIdx += idx
	
	numStart := eqIdx + 1
	for numStart < len(fullOutput) && (fullOutput[numStart] == ' ' || fullOutput[numStart] == '\n' || fullOutput[numStart] == '\t') {
		numStart++
	}
	
	numEnd := numStart
	for numEnd < len(fullOutput) && (strings.ContainsRune("0123456789.e-", rune(fullOutput[numEnd]))) {
		numEnd++
	}
	
	if numEnd > numStart {
		return fullOutput[numStart:numEnd]
	}
	return ""
}

func fallbackString(current, fallback string) string {
	current = strings.TrimSpace(current)
	fallback = strings.TrimSpace(fallback)
	if current == "" || current == "Unknown" {
		return fallback
	}
	return current
}

func fallbackInt(current, fallback int) int {
	if current == 0 {
		return fallback
	}
	return current
}

func fallbackFloat(current, fallback float64) float64 {
	if current == 0 {
		return fallback
	}
	return current
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(strings.TrimSpace(s))
	return i
}

func randomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	if _, err := crand.Read(buf); err != nil {
		panic(err)
	}
	for i := range buf {
		buf[i] = charset[int(buf[i])%len(charset)]
	}
	return string(buf)
}

func createVacFile(flacPath, outPath, totp string, key []byte, title, artist, album string, duration float64, bitrate, sampleRate, track int) error {
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	header := createHeader(totp, title, artist, album, duration, bitrate, sampleRate, track)
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

func createHeader(totp, title, artist, album string, duration float64, bitrate, sampleRate, track int) []byte {
	buf := make([]byte, headerSize)

	copy(buf[0:4], []byte(magicNumber))
	binary.BigEndian.PutUint32(buf[4:8], versionNumber)
	binary.BigEndian.PutUint32(buf[8:12], keySizeBytes)

	offset := 12
	copy(buf[offset:offset+totpSecretLength], padOrTrim(totp, totpSecretLength))
	offset += totpSecretLength
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

func play(path string) {
    keyBytes := sha256.Sum256([]byte(passphrase))

    // Open VAC file
    f, err := os.Open(path)
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    // Read & validate header
    if err := readAndValidateHeader(f); err != nil {
        log.Fatal(err)
    }

    // Read encrypted FLAC data
    encryptedData, err := io.ReadAll(f)
    if err != nil {
        log.Fatal(err)
    }

    // Decrypt FLAC bytes
    flacData, err := decrypt(encryptedData, keyBytes[:])
    if err != nil {
        log.Fatal("Decryption failed:", err)
    }

    // Start ffplay reading from stdin
    cmd := exec.Command("ffplay",
        "-i", "pipe:0", // read FLAC from stdin
        "-nodisp",      // audio only
        "-autoexit",    // exit when done
    )

    stdin, err := cmd.StdinPipe()
    if err != nil {
        log.Fatal(err)
    }
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        log.Fatal(err)
    }

    // Write decrypted FLAC directly to ffplay
    go func() {
        defer stdin.Close()
        if _, err := stdin.Write(flacData); err != nil {
            log.Fatal("Failed to stream FLAC to ffplay:", err)
        }
    }()

    if err := cmd.Wait(); err != nil {
        log.Fatal("ffplay failed:", err)
    }
}



func info(path string) {
	fi, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("VAC File: %s (%.2f MB)\n", path, float64(fi.Size())/1024/1024)

	f, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	header := make([]byte, headerSize)
	if _, err := io.ReadFull(f, header); err != nil {
		log.Fatal(err)
	}

	if string(header[0:4]) != magicNumber {
		log.Fatal("Not a VAC file")
	}

	offset := 12 + totpSecretLength
	title := strings.TrimRight(string(header[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	artist := strings.TrimRight(string(header[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	album := strings.TrimRight(string(header[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	duration := math.Float64frombits(binary.BigEndian.Uint64(header[offset : offset+8]))
	offset += 8
	bitrate := binary.BigEndian.Uint32(header[offset : offset+4])
	offset += 4
	sampleRate := binary.BigEndian.Uint32(header[offset : offset+4])
	offset += 4
	track := binary.BigEndian.Uint32(header[offset : offset+4])

	fmt.Printf("Title: %s\nArtist: %s\nAlbum: %s\n", title, artist, album)
	fmt.Printf("Duration: %.2fs\nBitrate: %d\nSampleRate: %d\nTrack: %d\n",
		duration, bitrate, sampleRate, track)
	fmt.Printf("Encrypted Audio Size: %.2f MB\n", float64(fi.Size()-headerSize)/1024/1024)
}

func readAndValidateHeader(r io.Reader) error {
	header := make([]byte, headerSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return err
	}

	if string(header[0:4]) != magicNumber {
		return fmt.Errorf("bad magic number")
	}

	version := binary.BigEndian.Uint32(header[4:8])
	keySize := binary.BigEndian.Uint32(header[8:12])

	if version != versionNumber {
		return fmt.Errorf("unsupported version %d", version)
	}
	if keySize != keySizeBytes {
		return fmt.Errorf("unexpected key size %d", keySize)
	}

	fmt.Printf("Playing VAC v%d (key size: %d bytes)\n", version, keySize)
	return nil
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
	if _, err := crand.Read(nonce); err != nil {
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
