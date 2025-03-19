package main

import (
	"fmt"
	"log"
	"math/rand"
	"meetai/ollama"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

func main() {

	// Generate a unique filename with timestamp
	filename := fmt.Sprintf("meeting_%s.mp3", time.Now().Format("20060102_150405"))
	filepath := "recordings/" + filename
	// Start recording
	recordCmd := startRecording(filename)
	defer func() {
		fmt.Println("Stopping recording...")
		recordCmd.Process.Signal(os.Interrupt)
		// fmt.Println("Waiting for recording to stop...")
		recordCmd.Wait()
		fmt.Println("Recording stopped")
	}()

	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Failed to start Playwright: %v", err)
	}
	defer pw.Stop()

	// Launch browser
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false), // Set to true for background execution
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--use-fake-ui-for-media-stream",
			"--use-fake-device-for-media-stream",
		},
	})
	if err != nil {
		log.Fatalf("Failed to launch browser: %v", err)
	}
	defer browser.Close()

	// Create new page
	page, err := browser.NewPage()
	if err != nil {
		log.Fatalf("Failed to create page: %v", err)
	}

	// Meeting configuration
	meetingURL := "https://meet.google.com/fho-nohr-kdg" // Replace with actual meeting link
	guestName := "clavirion"                             // Set guest name
	meetingDuration := 10 * time.Minute                  // Set meeting duration

	fmt.Printf("Joining meeting: %s as %s\n", meetingURL, guestName)

	// Navigate to the meeting URL
	if _, err := page.Goto(meetingURL); err != nil {
		log.Fatalf("Failed to navigate to meeting URL: %v", err)
	}

	// Add random delay to simulate human behavior
	randomDelay(2, 5)

	// Add realistic user behavior
	simulateHumanBehavior(page)

	// Join the meeting
	if err := joinMeeting(page, guestName); err != nil {
		log.Printf("Error joining meeting: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go monitorMeetingEnd(page, recordCmd, filepath, &wg)
	// Set up browser disconnect handler
	setupDisconnectHandler(browser, recordCmd)

	// Wait for the meeting duration
	fmt.Printf("Staying in meeting for %v\n", meetingDuration)
	// time.Sleep(meetingDuration)

	fmt.Println("Meeting time completed. Exiting...")
	wg.Wait()
}

// simulateHumanBehavior adds random mouse movements and scrolling to appear more human-like
func simulateHumanBehavior(page playwright.Page) {
	page.Mouse().Move(100+float64(rand.Intn(300)), 100+float64(rand.Intn(200)))
	page.Mouse().Wheel(0, 100)
	randomDelay(1, 2)
}

// joinMeeting handles the process of joining a Google Meet
func joinMeeting(page playwright.Page, guestName string) error {
	// Fill in name if the field is available
	nameInput := page.Locator("input[aria-label='Your name']")
	if nameInput != nil {
		isVisible, err := nameInput.IsVisible()
		if err == nil && isVisible {
			if err := nameInput.Fill(guestName); err != nil {
				log.Printf("Could not fill name: %v", err)
			} else {
				fmt.Println("Entered guest name")
			}
		}
	}

	// Click "Got it" button if visible
	handleButton(page, "button:has-text('Got it')", "Got it")

	// Ensure microphone and camera are off
	handleButton(page, "[aria-label='Turn off microphone']", "Turn off microphone")
	handleButton(page, "[aria-label='Turn off camera']", "Turn off camera")

	// Try to join the meeting - first try "Join now" button
	if !handleButton(page, "button:has-text('Join now')", "Join now") {
		// If "Join now" failed, try "Ask to join" button
		if !handleButton(page, "button:has-text('Ask to join')", "Ask to join") {
			return fmt.Errorf("could not find any join button")
		}
	}

	fmt.Println("Successfully requested to join the meeting")
	return nil
}

// handleButton attempts to click a button identified by selector
func handleButton(page playwright.Page, selector string, buttonName string) bool {
	button := page.Locator(selector)
	if button == nil {
		return false
	}

	isVisible, err := button.IsVisible()
	if err != nil || !isVisible {
		return false
	}

	if err := button.Click(); err != nil {
		log.Printf("Could not click %s button: %v", buttonName, err)
		return false
	}

	fmt.Printf("Clicked %s button\n", buttonName)
	randomDelay(1, 3)
	return true
}

// setupDisconnectHandler creates a handler for browser disconnects
func setupDisconnectHandler(browser playwright.Browser, recordCmd *exec.Cmd) {
	go func() {
		browser.On("disconnected", func() {
			fmt.Println("Browser closed unexpectedly. Stopping recording...")
			recordCmd.Process.Signal(os.Interrupt)
			recordCmd.Wait()
		})
	}()
}

// startRecording starts the FFmpeg process to record the meeting audio
func startRecording(filename string) *exec.Cmd {
	// Ensure the recordings folder exists
	recordingFolder := "recordings"
	if _, err := os.Stat(recordingFolder); os.IsNotExist(err) {
		os.Mkdir(recordingFolder, os.ModePerm)
	}

	// Generate the file path for the recording/audio file
	filepath := filepath.Join(recordingFolder, filename)

	// FFmpeg command to record audio
	cmd := exec.Command("ffmpeg",
		"-f", "dshow",
		"-i", "audio=CABLE Output (VB-Audio Virtual Cable)",
		"-ac", "2", // Stereo
		"-ar", "44100", // Sample rate 44.1kHz
		"-c:a", "libmp3lame", // MP3 codec
		"-b:a", "192k", // Bitrate 192kbps
		filepath,
	)

	// Capture FFmpeg output for debugging
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start FFmpeg recording: %v", err)
	}

	fmt.Println("Recording started:", filepath)
	return cmd
}

// randomDelay adds a random delay between actions to simulate human behavior
func randomDelay(min, max int) {
	delay := min + rand.Intn(max-min+1)
	time.Sleep(time.Duration(delay) * time.Second)
}

// monitorMeetingEnd continuously checks if the user has left the meeting
func monitorMeetingEnd(page playwright.Page, recordCmd *exec.Cmd, audioFilePath string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		// Check for meeting exit indicators
		leftMeetingText := page.Locator("text='You have left the meeting'")
		noOneElseText := page.Locator("text='No one else is in the meeting'")
		rejoinButton := page.Locator("button:has-text('Rejoin')")
		returnHomeButton := page.Locator("button:has-text('Return to home screen')")

		if isElementVisible(leftMeetingText) || isElementVisible(noOneElseText) || isElementVisible(rejoinButton) || isElementVisible(returnHomeButton) {
			fmt.Println("Meeting ended. Stopping recording...")
			// Stop recording
			stopRecording(recordCmd)
			// Close browser
			page.Close()

			// Transcribe the audio
			transcript, err := transcribeAudio(audioFilePath)
			if err != nil {
				fmt.Println("Error transcribing audio:", err)
				return
			}

			// Summarize the transcription
			summary, err := ollama.RunOllama(transcript)
			if err != nil {
				fmt.Println("Error summarizing text:", err)
				return
			}

			// Ensure the summaries folder exists
			summaryFolder := "summaries"
			if err := os.MkdirAll(summaryFolder, os.ModePerm); err != nil {
				fmt.Println("Error creating summaries folder:", err)
				return
			}

			// Generate the summary file path
			filename := filepath.Base(audioFilePath)                                           // Extract filename from path
			summaryFilePath := filepath.Join(summaryFolder, filename[:len(filename)-4]+".txt") // Replace .mp3 with .txt

			// Save the summary as a text file
			err = os.WriteFile(summaryFilePath, []byte(summary), 0644)
			if err != nil {
				fmt.Println("Error saving summary file:", err)
				return
			}

			fmt.Println("Summary saved at:", summaryFilePath)
			break
		}
		time.Sleep(2 * time.Second) // Check again after a delay
	}
}

// stopRecording stops the FFmpeg process properly on Windows
func stopRecording(recordCmd *exec.Cmd) {
	fmt.Println("Stopping recording gracefully...")
	err := recordCmd.Process.Signal(os.Interrupt)
	if recordCmd.Process != nil {
		if err := recordCmd.Process.Kill(); err != nil {
			fmt.Printf("Failed to kill FFmpeg process: %v\n", err)
		} else {
			fmt.Println("Recording process killed.")
		}
	}
	if err != nil {
		log.Printf("Failed to stop recording gracefully: %v", err)
	}

	// If CTRL+Break doesn't work, force kill the process
	// fmt.Println("Force stopping recording...")
	// if err := recordCmd.Process.Kill(); err != nil {
	// 	log.Printf("Failed to force kill recording: %v", err)
	// }
}

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

// isElementVisible checks if a Playwright locator is visible
func isElementVisible(locator playwright.Locator) bool {
	if locator == nil {
		return false
	}
	visible, err := locator.IsVisible()
	return err == nil && visible
}
