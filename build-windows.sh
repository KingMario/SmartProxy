#!/bin/bash
set -e

APP_NAME="SmartProxy"
OUT="${APP_NAME}.exe"
GOOS_TARGET="windows"
GOARCH_TARGET="amd64"

echo "🚧 Building ${OUT} (${GOOS_TARGET}/${GOARCH_TARGET}, windowsgui)..."

GOOS=${GOOS_TARGET} GOARCH=${GOARCH_TARGET} go build -ldflags="-H=windowsgui" -o "${OUT}" .

if [ -f "${OUT}" ]; then
    echo "✅ Build successful!"
    echo "📂 Binary placed at: ${OUT}"
else
    echo "❌ Build failed."
    exit 1
fi
