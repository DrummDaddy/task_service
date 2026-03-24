
FROM golang:1.25-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates && update-ca-certificates


COPY go.mod go.sum ./
RUN go mod download


COPY . .


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/task-service ./cmd


FROM alpine:3.19
RUN apk add --no-cache ca-certificates && update-ca-certificates
RUN adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=builder /out/task-service /app/task-service

EXPOSE 8080
USER appuser
ENTRYPOINT ["/app/task-service"]
