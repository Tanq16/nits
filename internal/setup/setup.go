package setup

import (
	"os/exec"
	"runtime"
)

type ToolStatus struct {
	Name    string
	Command string
	Found   bool
}

func CheckTools() []ToolStatus {
	return []ToolStatus{
		checkImageMagick(),
		checkFFProbe(),
		checkFFmpeg(),
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

func checkFFmpeg() ToolStatus {
	_, err := exec.LookPath("ffmpeg")
	return ToolStatus{Name: "FFmpeg", Command: "ffmpeg", Found: err == nil}
}
