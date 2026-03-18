#!/bin/bash
set -e

APP_NAME="SmartProxy"
APP_BUNDLE="${APP_NAME}.app"
BINARY_DEST="${APP_BUNDLE}/Contents/MacOS/${APP_NAME}"
RESOURCE_DIR="${APP_BUNDLE}/Contents/Resources"

echo "🚧 Building ${APP_NAME}..."

# Ensure directories exist
mkdir -p "${APP_BUNDLE}/Contents/MacOS"
mkdir -p "${RESOURCE_DIR}"

# Copy gfwlist.txt to Resources (since the app looks for it there)
if [ -f "gfwlist.txt" ]; then
    cp "gfwlist.txt" "${RESOURCE_DIR}/"
fi

# Build the binary (it will now include embedded assets)
go build -o "${BINARY_DEST}" .

if [ -f "${BINARY_DEST}" ]; then
    echo "✅ Build successful!"
    echo "📂 App Bundle: ${APP_BUNDLE}"
    chmod +x "${BINARY_DEST}"
    touch "${APP_BUNDLE}"
else
    echo "❌ Build failed."
    exit 1
fi
