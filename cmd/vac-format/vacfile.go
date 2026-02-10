package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

type VacHeader struct {
	title, artist, album string
	duration    float64
	bitrate, sampleRate, track int
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

func readHeader(path string) (*VacHeader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	header := make([]byte, headerSize)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil, err
	}
	
	if string(header[:4]) != magicNumber {
		return nil, fmt.Errorf("not a VAC file")
	}

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

