package main

import (
	"fmt"
	"log"
	"math/rand"
	"meetai/ollama"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

const (
	recordingFolder  = "recordings"
	transcriptFolder = "transcripts"
	summaryFolder    = "summaries"
)

func RunMeetingBot(meetingURL, guestName string) error {
	// Generate a unique filename with timestamp
	filename := fmt.Sprintf("meeting_%s.mp3", time.Now().Format("20060102_150405"))
	audioFilePath := filepath.Join(recordingFolder, filename)

	// Ensure recording directory exists
	if err := os.MkdirAll(recordingFolder, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create recording directory: %v", err)
	}

	// Start recording
	recordCmd := startRecording(audioFilePath)
	defer stopRecordingGracefully(recordCmd)

	// Initialize Playwright
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("failed to start Playwright: %v", err)
	}
	defer pw.Stop()

	// Launch browser
	browser, err := launchBrowser(pw)
	if err != nil {
		return fmt.Errorf("failed to launch browser: %v", err)
	}
	defer browser.Close()

	// Create new page
	page, err := browser.NewPage()
	if err != nil {
		return fmt.Errorf("failed to create page: %v", err)
	}

	fmt.Printf("Joining meeting: %s as %s\n", meetingURL, guestName)

	// Navigate to the meeting URL
	if _, err := page.Goto(meetingURL); err != nil {
		return fmt.Errorf("failed to navigate to meeting URL: %v", err)
	}

	// Add random delay and simulate human behavior
	randomDelay(2, 5)
	simulateHumanBehavior(page)

	// Join the meeting
	if err := joinMeeting(page, guestName); err != nil {
		log.Printf("Error joining meeting: %v", err)
	}

	// Wait for meeting to end
	var wg sync.WaitGroup
	wg.Add(1)
	go monitorMeetingEnd(page, recordCmd, audioFilePath, &wg)
	wg.Wait()

	return nil
}

func launchBrowser(pw *playwright.Playwright) (playwright.Browser, error) {
	return pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--use-fake-ui-for-media-stream",
			"--use-fake-device-for-media-stream",
			"--autoplay-policy=no-user-gesture-required",
		},
	})
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

	// Try to join the meeting
	if !handleButton(page, "button:has-text('Join now')", "Join now") {
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

// startRecording starts the FFmpeg process to record the meeting audio
func startRecording(filepath string) *exec.Cmd {
	cmd := exec.Command("ffmpeg",
		"-f", "dshow",
		"-i", "audio=CABLE Output (VB-Audio Virtual Cable)",
		"-ac", "2",
		"-ar", "44100",
		"-c:a", "libmp3lame",
		"-b:a", "192k",
		filepath,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start FFmpeg recording: %v", err)
	}

	fmt.Println("Recording started:", filepath)
	return cmd
}

// stopRecordingGracefully properly stops the FFmpeg recording process
func stopRecordingGracefully(recordCmd *exec.Cmd) {
	fmt.Println("Stopping recording...")
	if recordCmd.Process != nil {
		recordCmd.Process.Signal(os.Interrupt)
		recordCmd.Wait()
		fmt.Println("Recording stopped")
	}
}

// randomDelay adds a random delay between actions to simulate human behavior
func randomDelay(min, max int) {
	delay := min + rand.Intn(max-min+1)
	time.Sleep(time.Duration(delay) * time.Second)
}

// monitorMeetingEnd continuously checks if the user has left the meeting
func monitorMeetingEnd(page playwright.Page, recordCmd *exec.Cmd, audioFilePath string, wg *sync.WaitGroup) {
	defer wg.Done()

	// Target person information
	targetPerson := "basits@zenithive.com" // Email of the person we're tracking
	targetPersonName := "Basit Saiyed"     // Name of the person we're tracking

	// Track how long we've been in the meeting after the target person has left
	targetPersonLeftTime := time.Time{}
	exitTimeoutAfterTargetLeaves := 20 * time.Second

	for {
		// Check for meeting exit indicators
		exitIndicators := []string{
			"text='You have left the meeting'",
			"text='No one else is in the meeting'",
			"button:has-text('Rejoin')",
			"button:has-text('Return to home screen')",
		}

		for _, indicator := range exitIndicators {
			if isElementVisible(page.Locator(indicator)) {
				fmt.Println("Meeting ended. Stopping recording...")
				stopRecording(recordCmd)
				page.Close()

				processRecording(audioFilePath)
				return
			}
		}
		// Check if target person is still in the meeting
		targetPresent := isPersonInMeeting(page, targetPerson, targetPersonName)
		if !targetPresent {
			// Target person is not in the meeting
			if targetPersonLeftTime.IsZero() {
				// First detection of target person absence, start timer
				targetPersonLeftTime = time.Now()
				fmt.Printf("Target person %s (%s) has left the meeting. Starting exit timer.\n",
					targetPersonName, targetPerson)
			} else if time.Since(targetPersonLeftTime) > exitTimeoutAfterTargetLeaves {
				// We've waited long enough after target person left, now exit
				fmt.Printf("It's been %v since target person left. Leaving the meeting.\n",
					exitTimeoutAfterTargetLeaves)
				leaveCurrentMeeting(page)
				stopRecording(recordCmd)
				page.Close()

				processRecording(audioFilePath)
				return
			}
		} else {
			// Target person is back in the meeting, reset timer if needed
			if !targetPersonLeftTime.IsZero() {
				fmt.Printf("Target person %s (%s) is back in the meeting. Resetting exit timer.\n",
					targetPersonName, targetPerson)
				targetPersonLeftTime = time.Time{}
			}
		}
		time.Sleep(2 * time.Second)
	}
}

// isPersonInMeeting checks if a specific person is present in the meeting
func isPersonInMeeting(page playwright.Page, personEmail string, personName string) bool {
	// Try to find the participant panel first (if not already open)
	openParticipantPanel(page)

	// Look for this person in the participants list by email or name
	participantSelectors := []string{
		// Check by email
		`[aria-label*="${personEmail}"]`,
		`text="${personEmail}"`,
		// Check by name
		`[aria-label*="${personName}"]`,
		`text="${personName}"`,
		// More general selectors that might contain the name or email
		`div[role="listitem"]:has-text("${personEmail}")`,
		`div[role="listitem"]:has-text("${personName}")`,
	}

	// Replace template values with actual values
	for i, selector := range participantSelectors {
		participantSelectors[i] = strings.Replace(selector, "${personEmail}", personEmail, -1)
		participantSelectors[i] = strings.Replace(selector, "${personName}", personName, -1)
	}

	// Try each selector
	for _, selector := range participantSelectors {
		element := page.Locator(selector)
		if element != nil {
			visible, err := element.IsVisible()
			if err == nil && visible {
				count, err := element.Count()
				if err == nil && count > 0 {
					return true
				}
			}
		}
	}

	 // Check approach 2: Look at active speaker indicators or other UI elements
    // This approach works even if we can't open the participants panel
    activeSpeakerSelectors := []string{
        // Look for the person's name in active speaker labels
        `[data-active-speaker-label*="${personName}"]`,
        // Look for their tile with name label
        `[aria-label*="${personName}"][role="img"]`,
        `[aria-label*="${personName}"][role="button"]`,
        // Look in chat messages (if they sent any)
        `[data-sender-name*="${personName}"]`,
    }
    
    // Replace template values with actual values
    for i, selector := range activeSpeakerSelectors {
        activeSpeakerSelectors[i] = strings.Replace(selector, "${personName}", personName, -1)
    }
    
    // Try each active speaker/participant indicator selector
    for _, selector := range activeSpeakerSelectors {
        element := page.Locator(selector)
        if element != nil {
            visible, err := element.IsVisible()
            if err == nil && visible {
                count, err := element.Count()
                if err == nil && count > 0 {
                    return true
                }
            }
        }
    }

	return false
}

// openParticipantPanel attempts to open the participants panel if not already open
func openParticipantPanel(page playwright.Page) {
	// First, try to dismiss any popups that might be blocking the UI
    dismissPopups(page)

	// Potential selectors for the participant panel button
	participantButtonSelectors := []string{
		`[aria-label="Show everyone"]`,
		`[aria-label="Participants"]`,
		`[aria-label="People"]`,
		`button[aria-label*="participant"]`,
		`[data-tooltip="Show everyone"]`,
	}

	for _, selector := range participantButtonSelectors {
		button := page.Locator(selector)
		if button != nil {
			visible, err := button.IsVisible()
			if err == nil && visible {
				// Check if panel is already open
				panelOpenSelectors := []string{
					`[aria-label="Participants panel"]`,
					`[aria-label="People panel"]`,
					`div[role="dialog"]:has-text("People")`,
				}

				panelAlreadyOpen := false
				for _, panelSelector := range panelOpenSelectors {
					panel := page.Locator(panelSelector)
					if panel != nil {
						panelVisible, err := panel.IsVisible()
						if err == nil && panelVisible {
							panelAlreadyOpen = true
							break
						}
					}
				}

				if !panelAlreadyOpen {
					// Click to open panel
					if err := button.Click(); err == nil {
						// fmt.Println("Opened participants panel")
						// Wait for panel to appear
						time.Sleep(1 * time.Second)
						return
					}
				} else {
					// Panel already open
					return
				}
			}
		}
	}
}

// dismissPopups handles any popups that might appear during the meeting
func dismissPopups(page playwright.Page) {
    // List of common popup dismiss button selectors
    dismissButtonSelectors := []string{
        // The "Got it" button from your screenshot
        `button:has-text("Got it")`,
        `text="Got it"`,
        // Other common popup buttons
        `button:has-text("Dismiss")`,
        `button:has-text("Close")`,
        `button:has-text("I understand")`,
        `button:has-text("No thanks")`,
        `button:has-text("Skip")`,
        `button:has-text("Not now")`,
        // Close icons
        `[aria-label="Close"]`,
        `[aria-label="Dismiss"]`,
    }
    
    for _, selector := range dismissButtonSelectors {
        button := page.Locator(selector)
        if button != nil {
            visible, err := button.IsVisible()
            if err == nil && visible {
                fmt.Printf("Found popup dismiss button: %s\n", selector)
                if err := button.Click(); err != nil {
                    fmt.Printf("Failed to click dismiss button: %v\n", err)
                } else {
                    // fmt.Println("Successfully dismissed popup")
                    // Wait a moment for the popup to disappear
                    time.Sleep(500 * time.Millisecond)
                }
            }
        }
    }
}


// leaveCurrentMeeting attempts to exit the meeting gracefully
func leaveCurrentMeeting(page playwright.Page) {
	// Click the hang up/leave meeting button
	leaveButtons := []string{
		"[aria-label='Leave call']",
		"button[aria-label*='leave']",
		"button[aria-label*='hang up']",
		"button[data-is-muted='leave-call']",
		// Add more potential selectors for the leave button
	}

	for _, buttonSelector := range leaveButtons {
		button := page.Locator(buttonSelector)
		if button != nil {
			visible, err := button.IsVisible()
			if err == nil && visible {
				fmt.Println("Clicking leave meeting button...")
				if err := button.Click(); err != nil {
					fmt.Printf("Failed to click leave button: %v\n", err)
				} else {
					fmt.Println("Successfully left the meeting")
					randomDelay(1, 2)
					return
				}
			}
		}
	}

	fmt.Println("Could not find leave meeting button. Closing page instead.")
}

// processRecording handles transcription and summarization of the audio file
func processRecording(audioFilePath string) {
	// Transcribe the audio
	transcript, err := transcribeAudio(audioFilePath)
	if err != nil {
		fmt.Println("Error transcribing audio:", err)
		return
	}

	// Save transcript
	if err := saveOutput(audioFilePath, transcriptFolder, transcript); err != nil {
		fmt.Println("Error saving transcript:", err)
		return
	}

	// Summarize the transcription
	summary, err := ollama.RunOllama(transcript)
	if err != nil {
		fmt.Println("Error summarizing text:", err)
		return
	}

	// Save summary
	if err := saveOutput(audioFilePath, summaryFolder, summary); err != nil {
		fmt.Println("Error saving summary:", err)
		return
	}
}

// saveOutput saves data to a file with the same base name as the audio file but in a different folder
func saveOutput(audioFilePath, folderName, content string) error {
	if err := os.MkdirAll(folderName, os.ModePerm); err != nil {
		return fmt.Errorf("error creating folder %s: %v", folderName, err)
	}

	// Generate the output file path by replacing .mp3 with .txt
	filename := filepath.Base(audioFilePath)
	outputFilePath := filepath.Join(folderName, filename[:len(filename)-4]+".txt")

	if err := os.WriteFile(outputFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("error saving file: %v", err)
	}

	fmt.Printf("File saved at: %s\n", outputFilePath)
	return nil
}

// stopRecording stops the FFmpeg process
func stopRecording(recordCmd *exec.Cmd) {
	if recordCmd.Process == nil {
		return
	}

	fmt.Println("Stopping recording gracefully...")
	recordCmd.Process.Signal(os.Interrupt)

	// Force kill if needed
	if err := recordCmd.Process.Kill(); err != nil {
		fmt.Printf("Failed to kill FFmpeg process: %v\n", err)
	} else {
		fmt.Println("Recording process stopped.")
	}
}

// transcribeAudio runs the Python transcription script
func transcribeAudio(filePath string) (string, error) {
	fmt.Println("Transcribing audio...")
	cmd := exec.Command("python3", "transcribe.py", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running transcription script: %v", err)
	}
	return string(output), nil
}

// isElementVisible checks if a Playwright locator is visible
func isElementVisible(locator playwright.Locator) bool {
	if locator == nil {
		return false
	}
	visible, err := locator.IsVisible()
	return err == nil && visible
}
