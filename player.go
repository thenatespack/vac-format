package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Player interface {
	Play(io.Reader) error
}

type FFPlayPlayer struct{}
type VLCPlayer struct{}
type MPVPlayer struct{}

func Play(path string, player Player) error {
	fmt.Printf("🔍 Using passphrase: '%s' (%d chars)\n", passphrase, len(passphrase))
	keyBytes := deriveKey([]byte(passphrase))
	fmt.Printf("🔑 Derived key: %x\n", keyBytes[:16])

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	fmt.Printf("📦 Total file: %d bytes\n", len(data))
	
	if len(data) < headerSize {
		return fmt.Errorf("file too short for header")
	}
	
	headerBytes := data[:headerSize]
	if string(headerBytes[:4]) != magicNumber {
		return fmt.Errorf("not a VAC file")
	}

	offset := 12
	title := strings.TrimRight(string(headerBytes[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	artist := strings.TrimRight(string(headerBytes[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	album := strings.TrimRight(string(headerBytes[offset:offset+strFieldLength]), "\x00")
	offset += strFieldLength
	encData := data[headerSize:]
	fmt.Printf("🔓 Decrypting %d bytes of encrypted data\n", len(encData))
	
	decrypted, err := decrypt(encData, keyBytes)
	if err != nil {
		return fmt.Errorf("audio decryption failed: %w", err)
	}

	fmt.Printf("✅ SUCCESS - Decrypted %d bytes\n", len(decrypted))
	fmt.Printf("🎵 Playing: %s - %s (%s)\n", title, artist, album)
	return player.Play(bytes.NewReader(decrypted))
}

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

