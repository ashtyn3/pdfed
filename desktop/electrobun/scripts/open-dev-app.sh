#!/usr/bin/env sh
set -eu

APP_PATH="$(ls -d build/dev-*/pdfed-dev.app 2>/dev/null | head -n 1 || true)"

if [ -z "$APP_PATH" ]; then
  APP_PATH="$(ls -d build/dev-*/pdfed-desktop-dev.app 2>/dev/null | head -n 1 || true)"
fi

if [ -z "$APP_PATH" ]; then
  echo "No dev app bundle found under build/dev-*/" >&2
  exit 1
fi

open "$APP_PATH"
