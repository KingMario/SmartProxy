#!/bin/bash
set -e

APP_NAME="SmartProxy"

build_windows_gui() {
  echo "🚧 Building Windows GUI (no console) version..."
  ./build-windows.sh
  echo "✅ Windows GUI build: ${APP_NAME}.exe"
}

build_macos_app() {
  if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "ℹ️  Skipping macOS .app build (requires running on macOS host)."
    return 0
  fi

  if [[ ! -x "./build.sh" ]]; then
    chmod +x ./build.sh
  fi

  echo "🚧 Building macOS app bundle..."
  ./build.sh
  echo "✅ macOS app build: ${APP_NAME}.app"
}

echo "🚀 Building all targets..."
build_windows_gui
build_macos_app

echo "🎉 All available targets built."
