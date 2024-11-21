FROM golang:1.23 as builder
ARG CGO_ENABLED=0
WORKDIR /app

COPY vendor ./
COPY go.mod go.sum ./
COPY . .

RUN go build -o rest ./cmd/rest/main.go
RUN go build -o storage ./cmd/storage/main.go

FROM scratch

COPY --from=builder /app/rest /rest
COPY --from=builder /app/storage /storage

ENTRYPOINT ["/api"]
