#!/bin/bash
set -euo pipefail

APP_NAME="SmartProxy"
OUT="${APP_NAME}.exe"
GOOS_TARGET="windows"
GOARCH_TARGET="amd64"
ICON_SOURCE="assets/tray.ico"
RESOURCE_FILE="${APP_NAME}.syso"

export PATH="$(go env GOPATH)/bin:$PATH"

echo "🎨 Building frontend..."
(
  cd frontend
  npm run build
)

if ! command -v rsrc >/dev/null 2>&1; then
  echo "📦 Installing rsrc tool for resource embedding..."
  go install github.com/akavel/rsrc@v0.10.2
fi

if [[ ! -f "${ICON_SOURCE}" ]]; then
  echo "❌ Icon not found: ${ICON_SOURCE}"
  exit 1
fi

trap 'rm -f "${RESOURCE_FILE}"' ERR EXIT

echo "🎨 Embedding icon from ${ICON_SOURCE} into ${RESOURCE_FILE}..."
rsrc -ico "${ICON_SOURCE}" -o "${RESOURCE_FILE}"

echo "🚧 Building ${OUT} (${GOOS_TARGET}/${GOARCH_TARGET}, windowsgui)..."
GOOS=${GOOS_TARGET} GOARCH=${GOARCH_TARGET} go build -ldflags="-H=windowsgui" -o "${OUT}" .

if [[ -f "${OUT}" ]]; then
  echo "✅ Build successful!"
  echo "📂 Binary placed at: ${OUT}"
else
  echo "❌ Build failed."
  exit 1
fi
