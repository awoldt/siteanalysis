FROM golang:1.27-bookworm
WORKDIR /app 
COPY go.mod go.sum ./
RUN go mod download
COPY . . 
RUN go build -o app .

CMD ["./app"]