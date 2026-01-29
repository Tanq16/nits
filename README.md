<div align="center">
  <!-- Logo can be added at .github/assets/logo.png -->
  <h1>nits</h1>

  <a href="https://github.com/tanq16/nits/actions/workflows/build.yml"><img alt="Build Workflow" src="https://github.com/tanq16/nits/actions/workflows/build.yml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/nits/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/nits"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

A collection of tiny tools and scripts packaged as a single Go binary.

Most of these are conversions from Python scripts used over time. Converting to Go makes them easily usable across systems without worrying about dependencies.

A more robust tool is [anbu](https://github.com/tanq16/anbu). As and when I find a script or tool implemented here more useful and more frequently used, I promote it to `anbu`.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Files | `file-organizer`, `file-unzipper`, `file-json-uniq` | File management and organization utilities |
| Images | `img-webp`, `img-dedup` | Image compression and duplicate detection |
| Video | `video-info` | Video file analysis using ffprobe |
| Diagrams | `mermaid-svg` | Web interface for Mermaid diagram to SVG conversion |
| System | `setup` | Check if required third-party tools are installed |

## Installation

### Binary

Download from [releases](https://github.com/tanq16/nits/releases):

```bash
# Linux/macOS
curl -sL https://github.com/tanq16/nits/releases/latest/download/nits-$(uname -s)-$(uname -m) -o nits
chmod +x nits
sudo mv nits /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/tanq16/nits
cd nits
make build-local
```

**Requirements:** Go v1.24+

## Usage

### File Management

#### `file-organizer`

Group files into directories based on base name (e.g., `goku_1.jpg`, `goku_2.jpg` → `goku/`).

```bash
nits file-organizer [--dry-run]
```

**Flags:**
- `--dry-run, -r` - Check without making changes

**Examples:**

```bash
# Organize files in current directory
nits file-organizer

# Preview changes without moving files
nits file-organizer --dry-run
```

#### `file-unzipper`

Unzip all zip files in the current directory, creating a directory for each. Flattens single-subdirectory zips.

```bash
nits file-unzipper [--uuid-names]
```

**Flags:**
- `--uuid-names, -u` - Rename directories and files to UUIDs

**Examples:**

```bash
# Unzip all zip files in CWD
nits file-unzipper

# Unzip with UUID naming
nits file-unzipper --uuid-names
```

#### `file-json-uniq`

Remove duplicate items from a JSON slice based on a key.

```bash
nits file-json-uniq <file> --path <path> --key <key>
```

**Flags:**
- `--path, -p` - Path to the slice in JSON (e.g., 'references')
- `--key, -k` - Key to use for uniqueness (e.g., 'url')

**Examples:**

```bash
# Remove duplicate references based on URL
nits file-json-uniq data.json --path references --key url
```

### Image Processing

#### `img-webp`

Compress all images in current directory to WebP format with quality optimization.

```bash
nits img-webp [--dry-run] [--workers N]
```

**Flags:**
- `--dry-run, -r` - Process images without deleting originals
- `--workers, -w` - Number of workers for parallel processing (default: 4)

**Examples:**

```bash
# Compress images to WebP
nits img-webp

# Preview compression without deleting originals
nits img-webp --dry-run

# Use 8 parallel workers
nits img-webp --workers 8
```

#### `img-dedup`

Find duplicate images in current directory using perceptual hashing.

```bash
nits img-dedup [--hamming-distance N] [--workers N]
```

**Flags:**
- `--hamming-distance, -d` - Maximum Hamming distance for duplicate detection (default: 10)
- `--workers, -w` - Number of workers for parallel processing (default: 4)

**Examples:**

```bash
# Find duplicate images
nits img-dedup

# Use stricter duplicate detection
nits img-dedup --hamming-distance 5
```

### Video Analysis

#### `video-info`

Display detailed information about a video file using ffprobe.

```bash
nits video-info <file>
```

**Examples:**

```bash
# Show video info
nits video-info movie.mp4
```

### Diagrams

#### `mermaid-svg`

Start a web interface for creating Mermaid diagrams and exporting them as SVG/PNG.

```bash
nits mermaid-svg [--port PORT]
```

**Flags:**
- `--port, -p` - Port to listen on (default: 8080)

**Examples:**

```bash
# Start Mermaid SVG server on default port
nits mermaid-svg

# Use custom port
nits mermaid-svg --port 9999
```

Then open `http://localhost:8080` in your browser to use the diagram editor.

### System

#### `setup`

Check if required third-party tools are installed (ImageMagick, ffprobe).

```bash
nits setup
```

## Tips and Notes

- Run `nits setup` to verify required third-party tools are installed
- Use `--debug` flag with any command for verbose logging
- The `mermaid-svg` command requires no external dependencies - assets are embedded
- Image commands require ImageMagick (`convert` or `magick`)
- Video commands require FFmpeg (`ffprobe`)
- There is no versioning for this project - releases are updated continuously
