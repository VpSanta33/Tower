#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUT="${1:-$ROOT_DIR/bin/tower}"

cd "$ROOT_DIR/web"
if [ ! -d node_modules ]; then
  npm install
fi
npm run build

cd "$ROOT_DIR"
scripts/embed-web.sh
go build -o "$OUT" ./api/tower.go
