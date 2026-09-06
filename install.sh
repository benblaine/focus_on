#!/bin/bash
set -e

PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_NAME="FocusOn"
APP_INSTALL_DIR="/Applications"
BIN_INSTALL_DIR="$HOME/bin"
CLI_BIN="$BIN_INSTALL_DIR/focuson"
CRON_TIME="00:00"

# --- Widget ----------------------------------------------------------------

echo "Building $APP_NAME..."
xcodebuild \
  -project "$PROJECT_DIR/$APP_NAME.xcodeproj" \
  -scheme "$APP_NAME" \
  -configuration Release \
  -derivedDataPath "$PROJECT_DIR/.build" \
  build

BUILT_APP="$PROJECT_DIR/.build/Build/Products/Release/$APP_NAME.app"

echo "Installing $APP_NAME to $APP_INSTALL_DIR..."
rm -rf "$APP_INSTALL_DIR/$APP_NAME.app"
cp -R "$BUILT_APP" "$APP_INSTALL_DIR/"

# Relaunch if already running
if pgrep -x "$APP_NAME" > /dev/null; then
  echo "Relaunching $APP_NAME..."
  pkill -x "$APP_NAME" || true
  sleep 0.5
fi

open "$APP_INSTALL_DIR/$APP_NAME.app"

# --- CLI ---------------------------------------------------------------

if ! command -v go > /dev/null; then
  echo "Warning: go not found — skipping the focuson CLI and its cron job." >&2
  echo "Done — $APP_NAME installed and launched."
  exit 0
fi

echo "Building focuson CLI..."
mkdir -p "$BIN_INSTALL_DIR"
( cd "$PROJECT_DIR/cli" && go build -o "$CLI_BIN" . )
echo "Installed focuson to $CLI_BIN"

case ":$PATH:" in
  *":$BIN_INSTALL_DIR:"*) ;;
  *) echo "Note: $BIN_INSTALL_DIR isn't on your PATH — add it to your shell profile to run 'focuson' directly." ;;
esac

# --- Daily sync cron job -----------------------------------------------

echo "Installing daily sync job (runs at $CRON_TIME)..."
"$CLI_BIN" cron install --time "$CRON_TIME"

echo "Done — $APP_NAME installed and launched, focuson CLI installed, daily sync scheduled for $CRON_TIME."
