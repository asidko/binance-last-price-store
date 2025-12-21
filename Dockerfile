FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates make sqlite curl

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/Makefile .

CMD ["./server"]
