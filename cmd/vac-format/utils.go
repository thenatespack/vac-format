package main

import (
	"fmt"
	"os"
	"os/exec"
)

func usage() {
	fmt.Println("Vitakrypt Audio Codec (VAC) CLI v1.0")
	fmt.Println("Commands:")
	fmt.Println("  vac encode -flac <file|folder> [-output <file|folder>] [-batch]")
	fmt.Println("  vac play <file.vac> [-player ffplay|vlc|mpv]")
	fmt.Println("  vac info <file.vac>")
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
