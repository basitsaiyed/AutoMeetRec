package main

import (
    "encoding/json"
    "log"
    "net/http"
)

type MeetingRequest struct {
    MeetingURL string `json:"meeting_url"`
    GuestName  string `json:"guest_name"`
}

func main() {
    http.HandleFunc("/start-meeting", handleStartMeeting)
    log.Println("API server running on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleStartMeeting(w http.ResponseWriter, r *http.Request) {
    var req MeetingRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    go func() {
        if err := RunMeetingBot(req.MeetingURL, req.GuestName); err != nil {
            log.Printf("Meeting bot error: %v", err)
        }
    }()
    w.Write([]byte("Meeting bot started. You will receive the summary when done."))
}