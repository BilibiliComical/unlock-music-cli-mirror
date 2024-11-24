#!/bin/bash
set -e

PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
  "windows/386"
)

DEST_DIR=${DEST_DIR:-"dist"}

for PLATFORM in "${PLATFORMS[@]}"; do
  GOOS=${PLATFORM%/*}
  GOARCH=${PLATFORM#*/}
  echo "Building for $GOOS/$GOARCH"

  FILENAME="um-$GOOS-$GOARCH"
  if [ "$GOOS" = "windows" ]; then
    FILENAME="$FILENAME.exe"
  fi

  GOOS=$GOOS GOARCH=$GOARCH go build -v \
    -o "${DEST_DIR}/${FILENAME}" \
    -ldflags "-s -w -X main.AppVersion=$(git describe --tags --always --dirty)" \
    ./cmd/um
done

cd "$DEST_DIR"
sha256sum um-* > sha256sums.txt