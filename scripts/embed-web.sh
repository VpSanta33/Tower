#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SRC_DIR="$ROOT_DIR/web/dist"
DST_DIR="$ROOT_DIR/api/internal/web/dist"

if [ ! -f "$SRC_DIR/index.html" ]; then
  echo "web/dist not found. Run npm run build in web first." >&2
  exit 1
fi

find "$DST_DIR" -mindepth 1 ! -name '.gitignore' ! -name 'placeholder.txt' -exec rm -rf {} +
cp -R "$SRC_DIR"/. "$DST_DIR"/
