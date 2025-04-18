# Use an official Golang image as a base
FROM golang:1.23-bullseye

# Set working directory inside the container
WORKDIR /app

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    python3-pip \
    npm \
    curl && \
    curl -fsSL https://deb.nodesource.com/setup_18.x | bash - && \
    apt-get install -y nodejs && \
    rm -rf /var/lib/apt/lists/*  # Cleanup to reduce image size

# Verify Node.js version
RUN node -v

# Install Playwright
RUN npm install -g playwright
RUN playwright install --with-deps

# Install Whisper AI
RUN pip install openai-whisper

# Copy the application source code
COPY . .

# Build the Go application
RUN go build -o AutoMeetRec main.go

# Set the command to run the application
CMD ["./AutoMeetRec"]
