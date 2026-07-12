#!/bin/sh
set -eu

REMOTE_HOST=${REMOTE_HOST:-lingmirror}
REMOTE_DIR=${REMOTE_BACKUP_DIR:-/var/lib/docker/volumes/multisell_db_backups/_data}
DEST_DIR=${BACKUP_PULL_DEST:-$HOME/Backups/LingMirror}
RETENTION_DAYS=${BACKUP_PULL_RETENTION_DAYS:-30}

mkdir -p "$DEST_DIR"
chmod 700 "$DEST_DIR"

latest=$(ssh -o BatchMode=yes -o ConnectTimeout=10 "$REMOTE_HOST" \
  "find '$REMOTE_DIR' -maxdepth 1 -type f -name '*.dump' -printf '%f\n' | sort | tail -n 1")
[ -n "$latest" ] || { echo "no remote backup found" >&2; exit 1; }

if [ -s "$DEST_DIR/$latest" ] && [ -s "$DEST_DIR/$latest.sha256" ]; then
  expected=$(awk '{print $1}' "$DEST_DIR/$latest.sha256")
  actual=$(shasum -a 256 "$DEST_DIR/$latest" | awk '{print $1}')
  [ "$actual" = "$expected" ] || { echo "existing offsite backup checksum mismatch" >&2; exit 1; }
  chmod 400 "$DEST_DIR/$latest" "$DEST_DIR/$latest.sha256"
  chflags uchg "$DEST_DIR/$latest" "$DEST_DIR/$latest.sha256"
  echo "offsite backup already present and verified: $DEST_DIR/$latest"
else
  tmp="$DEST_DIR/.${latest}.partial"
  trap 'rm -f "$tmp" "$tmp.sha256"' EXIT HUP INT TERM
  scp -q "$REMOTE_HOST:$REMOTE_DIR/$latest" "$tmp"
  scp -q "$REMOTE_HOST:$REMOTE_DIR/$latest.sha256" "$tmp.sha256"

  expected=$(awk '{print $1}' "$tmp.sha256")
  actual=$(shasum -a 256 "$tmp" | awk '{print $1}')
  [ "$actual" = "$expected" ] || { echo "backup checksum mismatch" >&2; exit 1; }

  mv "$tmp" "$DEST_DIR/$latest"
  mv "$tmp.sha256" "$DEST_DIR/$latest.sha256"
  chmod 400 "$DEST_DIR/$latest" "$DEST_DIR/$latest.sha256"
  chflags uchg "$DEST_DIR/$latest" "$DEST_DIR/$latest.sha256"
  echo "offsite backup pulled, verified, and protected: $DEST_DIR/$latest"
fi

find "$DEST_DIR" -type f \( -name '*.dump' -o -name '*.dump.sha256' \) -mtime "+$RETENTION_DAYS" \
  -exec chflags nouchg {} + -exec rm -f {} +
