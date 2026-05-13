#!/usr/bin/env bash
set -euo pipefail

/srv/file-converter/bin/file-converter-smoke
curl --fail --silent http://127.0.0.1:8081/healthz >/dev/null
curl --fail --silent http://127.0.0.1:8081/readyz >/dev/null
echo "smoke checks passed"

