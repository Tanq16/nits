# Markdown viewer assets

`js/`, `css/`, and `fonts/` are populated by `make assets` (see the repo `Makefile`)
and are gitignored. This file exists so `go:embed static` has something to embed
before assets are downloaded.
