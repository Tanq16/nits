<div align="center">
  <img src=".github/assets/logo.svg" alt="nits Logo" width="200">
  <h1>nits</h1>

  <a href="https://github.com/tanq16/nits/actions/workflows/release.yaml"><img alt="Build Workflow" src="https://github.com/tanq16/nits/actions/workflows/release.yaml/badge.svg"></a>&nbsp;<a href="https://github.com/tanq16/nits/releases"><img alt="GitHub Release" src="https://img.shields.io/github/v/release/tanq16/nits"></a><br><br>
  <a href="#capabilities">Capabilities</a> &bull; <a href="#install">Install</a> &bull; <a href="#usage">Usage</a> &bull; <a href="#notes">Notes</a>
</div>

---

nits is a local CLI toolkit for short, one-off operations packaged as a single Go binary.

Anbu is the self-hosted IT hub for secrets, machines, and SSH, and nits is the local CLI toolkit.

## Capabilities

| Category | Commands | Description |
|---|---|---|
| Files | `archive`, `rename`, `duplicates`, `file-unzipper` | Archives, regex bulk rename, file duplicates, zip flattening |
| Images | `img-dedup` | Duplicate image detection via perceptual hashing |
| Network | `download`, `github-release`, `http-server`, `ip-info`, `fs-sync` | HTTP downloads, GitHub release asset fetching, local file serving, IP lookup, bidirectional sync |
| Generators | `uuid`, `random-string`, `passphrase`, `time` | UUIDs, random strings, Diceware passphrases, and timestamp parsing and diffs |
| Data | `convert`, `neo4j` | Format conversions and Neo4j Cypher queries |

## Install

### Binary

Download directly from [Releases](https://github.com/tanq16/nits/releases). Binaries are available for AMD64 and ARM64 on Linux and macOS:

```bash
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')
curl -sL "https://github.com/tanq16/nits/releases/latest/download/nits-${OS}-${ARCH}" -o nits
chmod +x nits
sudo mv nits /usr/local/bin/
```

### Build from Source

Requires Go 1.27 or newer:

```bash
git clone https://github.com/tanq16/nits.git
cd nits
make build
```

## Usage

All commands support `--debug` for structured debug logging.

### Files

#### `archive`

Create or extract zip archives with optional regex filters and AES-GCM encryption. `create` has the alias `c` and `extract` has the alias `e`.

```bash
nits archive create ./src ./docs
nits archive c ./src -o backup.zip
nits archive create ./src --include '\.go$' --exclude '_test\.go$'
nits archive create ./src --bare
nits archive create ./src --encrypt
echo pw | nits archive create ./src --password -
nits archive extract archive.zip.enc --password pw
nits archive e backup.zip
nits archive extract backup.zip --bare
```

#### `rename`

Batch rename files or directories with regular expressions and capture groups.

```bash
nits rename 'old_(.*)' 'new_\1'
nits rename --directories 'old_(.*)' 'new_\1'
nits rename '(.*)\.(.*)' '\1_backup.\2'
nits rename 'image-(\d+).jpg' 'IMG_\1.jpeg' --dry-run
nits rename '(.*)' '\1_\uuid'
nits rename '(.*)\.(.*)' '\1_\suid.\2'
```

#### `duplicates` (alias: `dup`)

Find duplicate files by size and SHA-256 hash.

```bash
nits duplicates
nits dup --recursive
nits dup --delete
```

#### `file-unzipper`

Unzip all zip files in CWD, creating a directory for each and flattening single subdirectories.

```bash
nits file-unzipper
nits file-unzipper --uuid-names
```

### Images

#### `img-dedup`

Find duplicate images in CWD using perceptual hashing.

```bash
nits img-dedup
nits img-dedup --hamming-distance 5
nits img-dedup --workers 8
```

### Network

#### `download` (alias: `dl`)

Multi-connection HTTP download with automatic fallback to single connection and resume.

```bash
nits download https://example.com/file.tar.gz
nits dl https://example.com/file.tar.gz -o package.tar.gz
nits dl https://example.com/file.tar.gz -c 16
nits dl https://example.com/file.tar.gz -H "Authorization: Bearer token"
nits dl https://example.com/file.tar.gz --proxy http://127.0.0.1:8080
```

#### `github-release` (aliases: `ghr`, `ghrelease`)

Download assets from latest GitHub releases with platform auto-detection.

```bash
nits github-release tanq16/nits
nits ghr https://github.com/tanq16/nits
nits ghr tanq16/nits --asset nits-linux-amd64
nits ghr tanq16/nits --manual
nits ghr tanq16/nits -o nits.bin
```

#### `http-server`

Serve the current working directory over HTTP, with optional file upload support.

```bash
nits http-server
nits http-server -l 0.0.0.0:8080
nits http-server --upload
```

#### `ip-info` (alias: `ip`)

Display local network interfaces and public IP details.

```bash
nits ip-info
nits ip-info --ipv6
```

#### `fs-sync`

Bidirectional file synchronization over HTTP or HTTPS.

```bash
nits fs-sync serve --mode send -p 8080 -d ./data
nits fs-sync client http://localhost:8080 -d ./backup
```

### Generators

#### `passphrase`

Generate Diceware-style hyphenated phrases with one capital and one digit by default.

```bash
nits passphrase
nits passphrase -l 5
nits passphrase --simple
```

#### `uuid`

Generate UUID v7, UUID v4, or 18-character short identifiers.

```bash
nits uuid
nits uuid --v4
nits uuid --short
nits uuid --v4 --short
```

#### `random-string` (alias: `random`)

Generate cryptographic random strings.

```bash
nits random-string
nits random -l 32
nits random --hex
nits random --digits
nits random --alpha
nits random --all
```

#### `time` (alias: `t`)

Display current time, parse timestamps, and compute epoch differences.

```bash
nits time now
nits t parse "13 Apr 25 16:30 EDT"
nits t until "13 Apr 25 16:30 EDT"
nits t diff 1744192475 1744497775
nits t diff 1744192475
```

### Data

#### `convert` (alias: `c`)

Convert data between Docker run, Docker compose, URL encoding, and JWT decoding.

```bash
nits convert docker-compose "docker run -d -p 80:80 nginx"
nits convert compose-docker docker-compose.yml
nits convert url "Hello World"
nits convert urld "Hello%20World"
nits convert jwtd "$TOKEN"
```

#### `neo4j`

Execute a YAML list of Cypher queries against a Neo4j database. `--uri`, `--user`, and `--password` fall back to `NITS_NEO4J_URI`, `NITS_NEO4J_USER`, and `NITS_NEO4J_PASSWORD`.

```bash
nits neo4j --query-file ./queries.yaml
nits neo4j --query-file ./queries.yaml -o results.json
nits neo4j --write --query-file ./writes.yaml
```

## Notes

- **Archive compatibility**: Encrypted `.enc` archives created by nits and anbu are fully cross-compatible.
- **Perceptual hashing**: `img-dedup` runs in pure Go with no external ImageMagick dependency.
- **Logging**: Passing `--debug` enables structured zerolog console or JSON output.
- **Releases**: Releases are calculated automatically using `[major-release]` and `[minor-release]` commit markers.
