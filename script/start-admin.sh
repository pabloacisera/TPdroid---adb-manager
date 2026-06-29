#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../api_admin"
[ -f .env ] && set -a && source .env && set +a
[ -d venv ] && source venv/bin/activate
exec python3 -m uvicorn main:app --host 127.0.0.1 --port 8000
