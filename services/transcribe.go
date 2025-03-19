package services

import (
	"fmt"
	"os/exec"
)

func transcribeAudio(filePath string) (string, error) {
	fmt.Println("Transcribing audio...")
	cmd := exec.Command("python3", "transcribe.py", filePath) // Run Python script
	output, err := cmd.CombinedOutput()
	// fmt.Println("output", string(output)) // Get script output
	if err != nil {
		return "", fmt.Errorf("error running script: %v", err)
	}
	return string(output), nil // Convert output to string
}
