<div align="center">
  <h1>nits</h1>

  <a href="https://github.com/tanq16/nits/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/nits/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/nits/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/nits"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#installation">Installation</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#tips-and-notes">Tips & Notes</a>
</div>

---

A collection of tiny tools and scripts packaged as a single Go binary.

Most of these are conversions from Python scripts used over time. Converting to Go makes them easily usable across systems without worrying about dependencies.

A more robust tool is [anbu](https://github.com/tanq16/anbu). As and when I find a script or tool implemented here more useful and more frequently used, I promote it to `anbu`.

## Capabilities

| Category | Commands | Description |
|----------|----------|-------------|
| Files | `file-organizer`, `file-unzipper`, `file-json-uniq`, `manual-rename`/`mrename` | File management, organization, and interactive rename |
| Images | `img-webp`, `img-dedup` | Image compression and duplicate detection |
| Video | `video-optimize` / `video-opt` | Video size optimization (H.265 CPU, max 1080p, CRF 30, 8-bit SDR, HDR tone-mapping, interactive `--manual`) |
| Data | `convert`, `neo4j` | Format conversion and Neo4j Cypher queries |
| Productivity | `tasks` | Lightweight local task tracker with pending/done status |
| Diagrams | `mermaid-svg`, `markdown`/`md` | Mermaid SVG conversion and markdown viewer |
| Network | `fs-sync` | One-shot bidirectional file synchronization over HTTP/HTTPS |
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

**Requirements:** Go v1.26+

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

### Video Optimization

#### `video-optimize` / `video-opt`

Optimize a video file for maximum space reduction using H.265 CPU encoding (`libx265`, CRF 30, preset medium, 8-bit `yuv420p`). Videos $> 1080\text{p}$ (e.g. 4K, 1440p) are automatically downscaled to fit within $1920\times 1080$ preserving aspect ratio; lower resolutions are kept at native size without resampling. 10-bit HDR sources are automatically tone-mapped to 8-bit standard dynamic range (SDR) to prevent washed-out colors. Audio is encoded to transparent 128 kbps AAC stereo.

```bash
nits video-optimize <file> [--manual]
```

**Flags:**
- `--manual, -m` - Interactively configure CRF (22–34), target resolution, audio bitrate, and encoder speed preset via selection prompts

**Examples:**

```bash
# Optimize with default settings (CRF 30, max 1080p, 8-bit SDR)
nits video-optimize movie.mkv
nits video-opt clip.mp4

# Interactively choose quality, resolution, audio, and preset
nits video-optimize movie.mkv --manual
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

Check if required third-party tools are installed (ImageMagick, ffprobe, ffmpeg).

```bash
nits setup
```

### Rename & Convert

#### `manual-rename` / `mrename`

Interactively rename files and directories one by one, optionally including directories, hidden files, and extension changes.

```bash
nits manual-rename
nits mrename -d          # include directories
nits mrename -H          # include hidden
nits mrename -x          # allow extension changes
```

#### `convert`

Convert data between docker run/compose formats, URL encoding, and JWT decoding.

```bash
nits convert docker-compose "docker run ..."
nits convert compose-docker compose.yaml
nits convert url "Hello World"
nits convert urld "Hello%20World"
nits convert jwtd "$TOKEN"
```

#### `tasks`

Lightweight personal task tracking with pending/done status, stored at `~/.config/nits/tasks.json`.

```bash
nits tasks add
nits tasks list [--done] [--filter REGEX]
nits tasks done ID
nits tasks delete ID
```

#### `markdown` / `md`

Web-based markdown viewer with syntax highlighting and Mermaid diagram rendering.

```bash
nits markdown
nits md -l :3000
```

#### `fs-sync`

One-shot bidirectional file synchronization over HTTP/HTTPS.

```bash
nits fs-sync serve --mode send|receive -p 8080 -d DIR [--ignore] [-t] [--delete] [-r]
nits fs-sync client URL -d DIR [--ignore] [-k] [--delete] [-r]
```

#### `neo4j`

Execute inline or file-based Cypher queries against a Neo4j database.

```bash
nits neo4j -q "MATCH (n) RETURN n LIMIT 5"
nits neo4j --query-file ./queries.yaml -o results.json
nits neo4j --write -q "CREATE (n:Person {name: 'Alice'}) RETURN n"
```

## Tips and Notes

- Run `nits setup` to verify required third-party tools are installed
- Use `--debug` for structured zerolog output, or `--for-ai` for plain-text AI-friendly output (mutually exclusive)
- Build with `make build-local` (runs `make assets`) so embedded mermaid/markdown JS assets are present
- Image commands require ImageMagick (`convert` or `magick`)
- Video commands require FFmpeg (`ffprobe` and `ffmpeg`)
- Releases follow semantic versioning based on commit messages (`[major-release]`, `[minor-release]`)
