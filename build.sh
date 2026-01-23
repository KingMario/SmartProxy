#!/bin/bash
set -e

APP_NAME="SmartProxy"
APP_BUNDLE="${APP_NAME}.app"
BINARY_DEST="${APP_BUNDLE}/Contents/MacOS/${APP_NAME}"

echo "🚧 Building ${APP_NAME}..."

# Ensure the MacOS directory exists
mkdir -p "${APP_BUNDLE}/Contents/MacOS"

# Build the binary
# -o specifies the output path and name, effectively "renaming" it from the default
go build -o "${BINARY_DEST}" smart-proxy-gui.go

if [ -f "${BINARY_DEST}" ]; then
    echo "✅ Build successful!"
    echo "📂 Binary placed at: ${BINARY_DEST}"
    chmod +x "${BINARY_DEST}"
else
    echo "❌ Build failed."
    exit 1
fi
