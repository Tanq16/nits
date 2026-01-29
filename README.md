# nits

This is a single binary with a collection of scripts and tools that have been used a couple times or are irregularly useful from time to time.

Most of these are a conversion from Python scripts I've used overtime, but converting to Go makes them easily usable across systems without worrying about dependencies in general.

A more robust tool is [anbu](https://github.com/tanq16/anbu). As and when I find a script or tool implemented here more useful and more frequently used, I promote it to `anbu`.

## Usage

Download the binary from releases and run it. The same release is updated over and over, there is no versioning for this project, unlike in `anbu`. To check if required third-party tools are installed, run `nits setup`.

To build locally, run:
```bash
git clone https://github.com/tanq16/nits && \
cd nits && \
go build .
```

Use Go v1.24+.

## Functionalities

```
file-json-uniq Remove duplicate items from a JSON slice based on a key
file-organizer Group files into dirs based on base name. eg. goku_1.jpg, goku_2.jpg -> goku/
file-unzipper  Unzip all zip files in the current directory
img-dedup      Find duplicate images in CWD using perceptual hashing
img-webp       Compress all images in CWD to WebP format with quality optimization
playground     Playground command for testing
setup          Check if required tools are installed
video-info     Display detailed information about a video file using ffprobe
```
