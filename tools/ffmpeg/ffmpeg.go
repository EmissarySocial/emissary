package ffmpeg

import "os/exec"

var isFFmpegInstalled = false

func init() {

	// Check to see if ffmpeg is installed
	_, err := exec.LookPath("ffmpeg")

	if err == nil {
		isFFmpegInstalled = true
	}
}
