import sys
import whisper
import warnings
warnings.filterwarnings("ignore")  # Suppresses warnings


def transcribe_audio(file_path):
    model = whisper.load_model("base")  # Load Whisper AI model
    result = model.transcribe(file_path)
    return result["text"]  # Return transcribed text

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python transcribe.py <audio_file>")
        sys.exit(1)

    audio_file = sys.argv[1]  # Get the file path from the command line
    transcription = transcribe_audio(audio_file)
    print(transcription)  # Print the transcription for Go to capture
