FROM --platform=$BUILDPLATFORM golang:1.20-alpine AS build

WORKDIR /build

## Copy go.mod and go.sum files, download dependencies so they are cached
COPY go.mod go.sum ./

RUN go mod download

# Copy sources
COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o message-handler ./cmd/messagehandler

FROM alpine

WORKDIR /app

COPY --from=build /build/message-handler /build/run/config.yaml ./
CMD ["./message-handler"]
