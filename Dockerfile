FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN go install github.com/pressly/goose/v3/cmd/goose@latest


FROM alpine:3.22

WORKDIR /app

COPY --from=builder /out/api ./api
COPY --from=builder /go/bin/goose /usr/local/bin/goose

COPY migrations ./migrations

EXPOSE 8080

CMD ["./api"]