# nits

This is a single binary with a collection of scripts and tools that have been used a couple times or are useful from time to time.

Most of these are basically conversion from a collection of Python scripts I've used overtime for quick things, but converting to Go makes them easily usable across systems without worrying about dependencies for the most part.

A more general purpose and more robust tool is [anbu](https://github.com/tanq16/anbu). As and when I find a script or quick tool implemented here more useful and more frequently used, I will promote them to go into `anbu`.

Running is as simple as downloading from releases and running. It's the same release that is updated over and over, so it's always latest despite time shown on github.

To check if required third-party tools are installed, run `nits setup`.

## Functionalities

- `file-organizer`: Organize files by grouping them into folders based on base name
- `file-unzipper`: Unzip all zip files in the current directory
- `img-dedup`: Find duplicate images in CWD using perceptual hashing
- `img-webp`: Compress all images in CWD to WebP format with quality optimization
- `playground`: Playground command for testing
- `setup`: Check if required tools are installed
