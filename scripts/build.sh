#!/usr/bin/env bash
set -euo pipefail

BUILD_DIR="bin"
BINARY_NAME="why"
VERSION="${VERSION:-1.0.0}"
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE} -s -w"

mkdir -p "${BUILD_DIR}"

echo "Building ${BINARY_NAME} (${VERSION})..."
CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${BINARY_NAME}" ./cmd/why

if [ "${1:-}" = "all" ]; then
    echo "Cross-compiling for Linux, macOS, and Windows..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${BINARY_NAME}-linux-amd64" ./cmd/why
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${BINARY_NAME}-linux-arm64" ./cmd/why
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${BINARY_NAME}-darwin-amd64" ./cmd/why
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${BINARY_NAME}-darwin-arm64" ./cmd/why
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe" ./cmd/why
    echo "Artifacts created in ${BUILD_DIR}/"
fi

echo "Done!"
