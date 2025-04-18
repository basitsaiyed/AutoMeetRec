FROM golang:1.23

WORKDIR /app
COPY . .

RUN go build -o meeting-bot-server server.go main.go

CMD ["./meeting-bot-server"]