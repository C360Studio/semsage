# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /semsage ./cmd/semsage

# Runtime stage
FROM alpine:3

RUN apk add --no-cache ca-certificates
COPY --from=builder /semsage /usr/local/bin/semsage
COPY configs/semsage.json /etc/semsage/semsage.json

EXPOSE 8090

ENTRYPOINT ["semsage"]
CMD ["-config", "/etc/semsage/semsage.json"]
