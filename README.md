# AutoMeetRec  

**AutoMeetRec** is an automated meeting recording, transcription, and summarization tool built with Golang. It captures meeting audio, transcribes it, and generates concise summaries using Ollama.  

## Features  
- 🎙️ **Automatic Meeting Recording** – Uses Playwright and FFmpeg to capture audio.  
- ✍️ **Transcription** – Converts recorded audio into text and saves it.  
- 📄 **Summarization** – Generates a concise summary of the transcribed text.  
- 📂 **Organized Storage** – Saves recordings, transcripts, and summaries in separate folders.  
- 🚀 **Efficient & Fast** – Optimized for performance and automation.  

## Installation  

### Install Dependencies  
Ensure you have **Go**, **FFmpeg**, and **Playwright** installed.  

- **Install Go**: [Download here](https://golang.org/dl/)  
- **Install FFmpeg**: [FFmpeg Installation Guide](https://ffmpeg.org/download.html)  
- **Install Playwright**:  
  ```sh
  npm install -g playwright
- **Install Python** (if not already installed):[Download From Microsoft Store](https://apps.microsoft.com/detail/9NCVDN91XZQP?hl=en-us&gl=IN&ocid=pdpshare)
  Check the installation:
  ```sh
  python3 --version
- **Install Whisper AI**: *in cmd
  ```sh
  pip install openai-whisper


1. **Clone the Repository**  
   ```sh
   git clone https://github.com/basitsaiyed/AutoMeetRec.git  
   cd AutoMeetRec
   
2. **Run the Application**  
   ```sh
   go run main.go

## Usage  
  - Start the application to begin recording meetings.
  - Once a meeting ends, the audio is automatically saved in the recordings/ folder.
  - The transcription process starts immediately after the recording is saved.
  - A transcript is generated and stored in the transcripts/ folder.
  - A summary of the transcript is generated and stored in the summaries/ folder.

## Folder Structure
AutoMeetRec/   
│── recordings/     
│── transcripts/    
│── summaries/    
│── main.go    
│── ollama/    
│   └── runOllama.go  
│── README.md       
  

## Contributing
  Contributions are welcome! Feel free to submit a pull request or open an issue.
