#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="/srv/file-converter"
BIN_DIR="$APP_ROOT/bin"

install -d "$BIN_DIR"
install -d "$APP_ROOT/data/uploads"
install -d "$APP_ROOT/data/outputs"
install -d "$APP_ROOT/data/tmp"

cp ./bin/file-converter "$BIN_DIR/file-converter"
cp ./bin/file-converter-migrate "$BIN_DIR/file-converter-migrate"
cp ./bin/file-converter-smoke "$BIN_DIR/file-converter-smoke"

"$BIN_DIR/file-converter-migrate" up
systemctl restart file-converter
systemctl status file-converter --no-pager
nginx -t
systemctl reload nginx
"$APP_ROOT/deploy/scripts/smoke-check.sh"
