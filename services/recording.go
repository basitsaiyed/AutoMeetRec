package services

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const RecordingFolder = "recordings"

func StartRecording() (string, *exec.Cmd, error) {
	if _, err := os.Stat(RecordingFolder); os.IsNotExist(err) {
		if err := os.MkdirAll(RecordingFolder, os.ModePerm); err != nil {
			return "", nil, fmt.Errorf("failed to create recordings directory: %v", err)
		}
	}

	timeStr := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("meeting_%s.mp3", timeStr)
	audioFilepath := filepath.Join(RecordingFolder, filename)

	cmd := exec.Command("ffmpeg",
		"-f", "dshow",
		"-i", "audio=CABLE Output (VB-Audio Virtual Cable)",
		"-ac", "2", 
		"-ar", "44100", 
		"-c:a", "libmp3lame", 
		"-b:a", "192k",
		audioFilepath,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return "", nil, fmt.Errorf("failed to start FFmpeg recording: %v", err)
	}

	log.Println("Recording started:", audioFilepath)
	return audioFilepath, cmd, nil
}

func StopRecording(cmd *exec.Cmd) {
	if cmd != nil {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
		time.Sleep(3 * time.Second) // Ensure FFmpeg finalizes the file
	}
}
