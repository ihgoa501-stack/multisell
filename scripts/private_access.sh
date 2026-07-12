#!/bin/sh
set -eu

SSH_HOST=${SSH_HOST:-lingmirror}
LOCAL_PORT=${LOCAL_PORT:-8088}

echo "LingMirror private access: http://127.0.0.1:${LOCAL_PORT}"
echo "Keep this terminal open; press Ctrl-C to close the encrypted tunnel."
exec ssh -o ExitOnForwardFailure=yes -N \
  -L "${LOCAL_PORT}:127.0.0.1:8088" "$SSH_HOST"
