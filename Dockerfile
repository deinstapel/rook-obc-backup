# build stage
FROM golang:1.21 as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

FROM alpine:3.18 as certs
RUN apk add ca-certificates && update-ca-certificates

# final stage
FROM minio/mc:RELEASE.2023-09-07T22-48-55Z
COPY --from=builder /app/rook-obc-backup /app/
ENTRYPOINT ["/app/rook-obc-backup"]
