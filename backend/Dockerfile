FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go run github.com/99designs/gqlgen generate
RUN CGO_ENABLED=0 GOOS=linux go build -o /splitty ./cmd/server

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /splitty /splitty
EXPOSE 8080
CMD ["/splitty"]
