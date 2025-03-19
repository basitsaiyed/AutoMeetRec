package ollama

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func RunOllama(transcribe string) (string, error) {
	cmd := exec.Command("ollama", "run", "llama3.2", "Strictly summarize this in 100 words \n"+transcribe)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {

		return "", fmt.Errorf("error running ollama: %v", err)
	}

	cleanOutput := cleanANSI(string(output))
	// fmt.Println("Output: ", cleanOutput)
	return cleanOutput, nil
}
func cleanANSI(input string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mGK]|\x1b\[[?]25[lh]|\x1b\[[?]2026[lh]|\x1b\[1G`)
	spinnerRegex := regexp.MustCompile(`[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]`)
	thinkTagRegex := regexp.MustCompile(`(?i)<think>\s*<\/think>\s*`)
	cleaned := ansiRegex.ReplaceAllString(input, "")
	cleaned = spinnerRegex.ReplaceAllString(cleaned, "")
	cleaned = thinkTagRegex.ReplaceAllString(cleaned, "")
	return strings.TrimSpace(cleaned)
}
