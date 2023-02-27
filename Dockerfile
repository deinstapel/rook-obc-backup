# build stage
FROM golang:1.20 as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

FROM alpine:3.17 as certs
RUN apk add ca-certificates && update-ca-certificates

# final stage
FROM minio/mc
COPY --from=builder /app/rook-obc-backup /app/
ENTRYPOINT ["/app/rook-obc-backup"]
