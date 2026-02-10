# Vitakrypt Audio Codec (VAC) CLI v1.0

Vitakrypt Audio Codec (VAC) is a lightweight command-line tool for encoding FLAC files into an encrypted VAC format, inspecting metadata, and playing them securely. This version uses single-passphrase AES-GCM encryption. It is designed for safe audio streaming, allowing users to download music without risking piracy.

---

## Features

- Encode FLAC files to encrypted `.vac` format.
- Batch encode entire folders of FLAC files.
- Play VAC files directly using `ffplay`, `vlc`, or `mpv`.
- View metadata of VAC files: title, artist, album, duration, bitrate, sample rate, and track number.
- Single-passphrase encryption (default: `"hello mario"`).
- Fully cross-platform (requires a media player installed).

---

## Installation

### Download Binary

Visit **[Releases](#)** to download the binary for your platform.

### Build From Source

```bash
git clone https://github.com/yourusername/vitakrypt-vac.git
cd vitakrypt-vac
go build -o vac main.go
````

Make sure `ffplay`, `vlc`, or `mpv` is installed and in your system PATH.

---

## Usage

```bash
vac <command> [options]
```

### 1. Encode FLAC to VAC

**Single file:**

```bash
vac encode -flac path/to/file.flac [-output path/to/file.vac] [-passphrase "yourpass"]
```

**Folder (batch mode):**

```bash
vac encode -flac path/to/folder -batch [-output path/to/folder] [-passphrase "yourpass"]
```

* If no output path is specified, VAC files are created in the same folder as input.
* Batch mode processes all `.flac` files recursively.

### 2. Play VAC Files

```bash
vac play path/to/file.vac [-player ffplay|vlc|mpv]
```

* Default player is automatically detected (`mpv > vlc > ffplay`).

### 3. View VAC File Info

```bash
vac info path/to/file.vac
```

* Outputs metadata like title, artist, album, duration, bitrate, and sample rate.

---

## Example Workflow

```bash
# Encode a single FLAC file
vac encode -flac "song.flac" -passphrase "supersecret"

# Batch encode all FLAC files in a folder
vac encode -flac "./albums" -batch -passphrase "supersecret"

# Play a VAC file with mpv
vac play "./albums/song.vac" -player mpv

# Display metadata of a VAC file
vac info "./albums/song.vac"
```

---

## VAC File Format

* **Magic number:** `CSNG` (4 bytes)
* **Version:** 1 (4 bytes)
* **Key size:** 32 bytes (AES-256)
* **Metadata fields:** title, artist, album (each 64 bytes)
* **Duration:** `float64` (8 bytes)
* **Bitrate, Sample Rate, Track:** 4 bytes each
* **Encrypted FLAC data** follows header (AES-GCM)

---

## Encryption

* **AES-256 GCM** with a passphrase-derived key (`SHA-512 → 32 bytes`).
* Nonce is randomly generated per file and prepended to ciphertext.
* Single-passphrase simplifies key management.

---

## Dependencies

* Go standard library
* `github.com/dhowden/tag` (for FLAC metadata)
* External players: `ffplay`, `vlc`, or `mpv`

**Notes:**

* VAC files can only be played by this CLI (or any app implementing the same header/decryption logic).
* Default passphrase is `"hello mario"` — strongly recommended to change it in production.

---

## Optional GUI / Terminal Player

If you prefer a **terminal-based interface** to browse, queue, and play `.vac` files interactively, you can use **CipherSong VAC Player**.

* Built with **Textual**, it allows you to:

  * Browse your VAC library
  * Enqueue and manage a playback queue
  * Play songs one-by-one or all in sequence
  * Skip or remove songs on-the-fly
* Integrates with the VAC CLI and supports **mpv, VLC, or ffplay**.

**Installation & Usage:**

```bash
git clone https://github.com/yourusername/ciphersong-vac-player.git
cd ciphersong-vac-player
pip install textual
python main.py <path_to_vac_library>
```

**GitHub Repository:** [CipherSong VAC Player](https://github.com/thenatespack/ciphersong-vac-player)

**Keyboard Shortcuts:**

* `q` – Quit
* `r` – Refresh library
* `e` – Enqueue selected song
* `d` – Delete from queue
* `n` – Play next
* `a` – Play all queued songs
* `s` – Skip current song

---

## License

GNU License


