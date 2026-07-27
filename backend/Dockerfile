# Go 1.21 is EOL — the alpine image is no longer available on Docker Hub.
# Use golang:1.24-alpine which is the latest LTS with migration-friendly defaults.
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && CGO_ENABLED=0 go build -o server .

FROM alpine:3.21
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
