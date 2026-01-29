package setup

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/tanq16/nits/internal/utils"
)

type ToolStatus struct {
	Name    string
	Command string
	Found   bool
}

func RunSetup() {
	imStatus := checkImageMagick()
	ffStatus := checkFFProbe()

	if imStatus.Found {
		utils.PrintSuccess(fmt.Sprintf("ImageMagick is installed (%s)", imStatus.Command))
		log.Debug().Str("package", "setup").Str("tool", "ImageMagick").Str("command", imStatus.Command).Msg("Tool check")
	} else {
		utils.PrintError(fmt.Sprintf("ImageMagick is not installed (expected: %s)", imStatus.Command), nil)
		log.Debug().Str("package", "setup").Str("tool", "ImageMagick").Str("command", imStatus.Command).Msg("Tool missing")
	}

	if ffStatus.Found {
		utils.PrintSuccess("FFProbe is installed")
		log.Debug().Str("package", "setup").Str("tool", "FFProbe").Msg("Tool check")
	} else {
		utils.PrintError("FFProbe is not installed", nil)
		log.Debug().Str("package", "setup").Str("tool", "FFProbe").Msg("Tool missing")
	}
}

func checkImageMagick() ToolStatus {
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
	return ToolStatus{Name: "ImageMagick", Command: cmdName, Found: found}
}

func checkFFProbe() ToolStatus {
	_, err := exec.LookPath("ffprobe")
	return ToolStatus{Name: "FFProbe", Command: "ffprobe", Found: err == nil}
}
