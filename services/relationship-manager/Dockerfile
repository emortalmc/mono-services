FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build

ARG SERVICE_NAME=relationship-manager

WORKDIR /build

# Copy sources
COPY . .

RUN go mod download

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o $SERVICE_NAME ./cmd/$SERVICE_NAME

FROM alpine

ARG SERVICE_NAME=relationship-manager

WORKDIR /app

COPY --from=build /build/$SERVICE_NAME /build/run ./
CMD ["./relationship-manager"]