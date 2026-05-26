#!/usr/bin/env bash
# Smoke-test the contract-management API.
#
# Post-cutover: this service does NOT issue its own tokens. The token is
# minted by the platform authentication service via OAuth2 password grant,
# then presented as Bearer. Usage:
#
#   ./scripts/smoke_test.sh \
#       http://localhost:4103/v1\
#       http://localhost:3001/oauth/token \
#       admin@iag.local \
#       changeme
#
# The four args default to local-dev values from deploy/docker-compose.yml.
set -euo pipefail

BASE="${1:-http://localhost:4103/v1}"
TOKEN_URL="${2:-http://localhost:3001/oauth/token}"
USERNAME="${3:-admin@iag.local}"
PASSWORD="${4:-changeme}"
PASS=0
FAIL=0
ACCESS_TOKEN=""

mint_token() {
  local resp
  resp=$(curl -s -X POST \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -d "grant_type=password&username=${USERNAME}&password=${PASSWORD}" \
    "$TOKEN_URL")
  ACCESS_TOKEN=$(echo "$resp" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
  if [ -z "$ACCESS_TOKEN" ]; then
    echo "FATAL: could not mint token from $TOKEN_URL"
    echo "  response: $resp"
    exit 1
  fi
  echo "Minted user token (TTL truncated)"
}

curl_auth() {
  curl "$@" -H "Authorization: Bearer $ACCESS_TOKEN"
}

check() {
  local method="$1" path="$2" expect="$3" body="${4:-}"
  local code
  if [ -n "$body" ]; then
    code=$(curl_auth -s -o /tmp/cm_body -w "%{http_code}" -X "$method" \
      -H "Content-Type: application/json" -d "$body" "$BASE$path")
  else
    code=$(curl_auth -s -o /tmp/cm_body -w "%{http_code}" -X "$method" "$BASE$path")
  fi
  if [ "$code" = "$expect" ]; then
    echo "OK  $method $path -> $code"
    PASS=$((PASS + 1))
  else
    echo "FAIL $method $path -> $code (want $expect)"
    head -c 200 /tmp/cm_body 2>/dev/null; echo
    FAIL=$((FAIL + 1))
  fi
}

echo "=== contract-management smoke test: $BASE ==="

# Public health
check GET /health 200
check GET /health/live 200
check GET /health/ready 200

# Mint token and run protected paths.
mint_token

check GET /bootstrap 200
check GET /auth/session 200

check GET /permissions/catalog 200
check GET /permissions/builtin 200
check GET /permissions/me 200
check POST /permissions/check 200 '{"keys":["contracts.read"]}'

check GET /workspace 200
check GET /frontend 200
check GET /contracts 200
check GET /zones 200
check GET /engineers 200
check GET /users 200
check GET /milestones 200
check GET /materials 200
check GET /projects 200
check GET /roles 200
check GET /audit 200
check GET /assistance 200
check GET /profile/photo 200
check GET /exports/contracts.csv 200

check POST /contracts 201 \
  '{"no":"C-TEST","name":"Smoke test","zone":"Z1","status":"Planning","pri":"Medium","prog":10,"sup":"James Okello","remarks":""}'
check GET /contracts/C-TEST 200
check PATCH /contracts/C-TEST 200 '{"prog":25}'
check DELETE /contracts/C-TEST 204

check POST /audit 201 '{"action":"Smoke test","detail":"automated"}'

TINY='data:image/png;base64,iVBORw0KGgo='
check PUT /profile/photo 200 "{\"email\":\"$USERNAME\",\"dataUrl\":\"$TINY\"}"
check POST /uploads/profile 200 "{\"email\":\"$USERNAME\",\"dataUrl\":\"$TINY\"}"

check PUT /insights/scan 204 '{"recordCount":1,"scannedAt":"2026-01-01"}'

echo ""
echo "Passed: $PASS  Failed: $FAIL"
[ "$FAIL" -eq 0 ]
