package setup

import (
	"os/exec"
	"runtime"

	"github.com/rs/zerolog/log"
)

func RunSetup() {
	checkImageMagick()
}

func checkImageMagick() {
	var cmdName string
	var found bool
	switch runtime.GOOS {
	case "windows":
		if _, err := exec.LookPath("magick.exe"); err == nil {
			cmdName = "magick.exe"
			found = true
		} else if _, err := exec.LookPath("magick"); err == nil {
			cmdName = "magick"
			found = true
		} else {
			cmdName = "magick"
			found = false
		}
	case "darwin":
		if _, err := exec.LookPath("convert"); err == nil {
			cmdName = "convert"
			found = true
		} else if _, err := exec.LookPath("magick"); err == nil {
			cmdName = "magick"
			found = true
		} else {
			cmdName = "convert"
			found = false
		}
	default:
		if _, err := exec.LookPath("convert"); err == nil {
			cmdName = "convert"
			found = true
		} else if _, err := exec.LookPath("magick"); err == nil {
			cmdName = "magick"
			found = true
		} else {
			cmdName = "convert"
			found = false
		}
	}
	if found {
		log.Info().Str("tool", "ImageMagick").Str("command", cmdName).Msg("ImageMagick is installed")
	} else {
		log.Error().Str("tool", "ImageMagick").Str("command", cmdName).Msg("ImageMagick is not installed")
	}
}
