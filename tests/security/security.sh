#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PROJECT_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)
FRONTEND_SRC="$PROJECT_ROOT/frontend/src"
[ -d "$FRONTEND_SRC" ] || { echo "SECURITY TEST FAILED: frontend/src was not found: $FRONTEND_SRC" >&2; exit 1; }

BASE_URL="${SECURITY_BASE_URL:-http://localhost:8080}"
USER_EMAIL="${SECURITY_USER_EMAIL:?SECURITY_USER_EMAIL is required}"
USER_PASSWORD="${SECURITY_USER_PASSWORD:?SECURITY_USER_PASSWORD is required}"
EVIL_ORIGIN="${SECURITY_EVIL_ORIGIN:-https://evil.example}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fail() { echo "SECURITY TEST FAILED: $*" >&2; exit 1; }
pass() { echo "ok: $*"; }
json_value() { python3 -c 'import json,sys; print(json.load(sys.stdin).get(sys.argv[1], ""))' "$1"; }

login() {
  jar="$1"; headers="$2"; body="$3"
  status=$(curl -sS -D "$headers" -c "$jar" -o "$body" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASSWORD\"}" \
    "$BASE_URL/api/auth/login")
  [ "$status" = "200" ] || fail "login returned HTTP $status: $(cat "$body")"
}

login "$TMP/cookies" "$TMP/login.headers" "$TMP/login.json"
ACCESS_TOKEN=$(cat "$TMP/login.json" | json_value access_token)
[ -n "$ACCESS_TOKEN" ] || fail "login did not return an access token"

# JWT tampering must be rejected.
TAMPERED="${ACCESS_TOKEN%?}x"
status=$(curl -sS -o "$TMP/tampered.json" -w '%{http_code}' -H "Authorization: Bearer $TAMPERED" "$BASE_URL/api/users/me")
[ "$status" = "401" ] || fail "tampered JWT returned HTTP $status"
pass "tampered JWT is rejected"

# Login cookies: refresh is HttpOnly; test environment uses Strict and non-production may omit Secure.
grep -i '^Set-Cookie: refresh_token=' "$TMP/login.headers" | grep -qi 'HttpOnly' || fail "refresh cookie is not HttpOnly"
grep -i '^Set-Cookie: refresh_token=' "$TMP/login.headers" | grep -qi 'SameSite=Strict' || fail "refresh cookie is not SameSite=Strict"
grep -i '^Set-Cookie: csrf_token=' "$TMP/login.headers" | grep -qi 'SameSite=Strict' || fail "CSRF cookie is not SameSite=Strict"
pass "cookie security attributes are present"

# CSRF: refresh with cookie but without matching X-CSRF-Token must fail.
status=$(curl -sS -b "$TMP/cookies" -o "$TMP/csrf.json" -w '%{http_code}' -X POST "$BASE_URL/api/auth/refresh")
[ "$status" = "403" ] || fail "refresh without CSRF token returned HTTP $status"
pass "CSRF is enforced on refresh"

# Refresh rotation/reuse: preserve old refresh+CSRF, rotate once, then reuse the old pair.
OLD_REFRESH=$(awk '$6=="refresh_token" {print $7}' "$TMP/cookies" | tail -1)
OLD_CSRF=$(awk '$6=="csrf_token" {print $7}' "$TMP/cookies" | tail -1)
[ -n "$OLD_REFRESH" ] && [ -n "$OLD_CSRF" ] || fail "auth cookies were not written"
printf '# Netscape HTTP Cookie File\nlocalhost\tFALSE\t/api/auth\tFALSE\t2147483647\trefresh_token\t%s\nlocalhost\tFALSE\t/\tFALSE\t2147483647\tcsrf_token\t%s\n' "$OLD_REFRESH" "$OLD_CSRF" > "$TMP/old.cookies"
status=$(curl -sS -b "$TMP/old.cookies" -c "$TMP/rotated.cookies" -o "$TMP/refresh1.json" -w '%{http_code}' \
  -X POST -H "X-CSRF-Token: $OLD_CSRF" "$BASE_URL/api/auth/refresh")
[ "$status" = "200" ] || fail "first refresh returned HTTP $status: $(cat "$TMP/refresh1.json")"
status=$(curl -sS -b "$TMP/old.cookies" -o "$TMP/reuse.json" -w '%{http_code}' \
  -X POST -H "X-CSRF-Token: $OLD_CSRF" "$BASE_URL/api/auth/refresh")
[ "$status" = "401" ] || fail "reused refresh token returned HTTP $status"
grep -q 'REFRESH_TOKEN_REUSED' "$TMP/reuse.json" || fail "refresh reuse error code is missing"
pass "refresh token reuse is detected"

# Login again for remaining authenticated checks because reuse invalidates the family.
login "$TMP/cookies2" "$TMP/login2.headers" "$TMP/login2.json"
ACCESS_TOKEN=$(cat "$TMP/login2.json" | json_value access_token)
CSRF=$(awk '$6=="csrf_token" {print $7}' "$TMP/cookies2" | tail -1)

# SQL injection payload must be rejected and must not leak SQL/stack details.
status=$(curl -sS -o "$TMP/sqli.json" -w '%{http_code}' \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  --get --data-urlencode "source=NORMAL' OR '1'='1" \
  "$BASE_URL/api/recommendation-history")
case "$status" in 400|404|422) ;; *) fail "SQL injection probe returned unexpected HTTP $status";; esac
if grep -Eqi 'SQLSTATE|SELECT[[:space:]].*FROM|stack trace|panic:' "$TMP/sqli.json"; then
  fail "SQL/stack information leaked in SQL injection response"
fi
pass "SQL injection probe is rejected without internal leakage"

# CORS must not reflect an untrusted Origin and must not use wildcard with credentials.
curl -sS -D "$TMP/cors.headers" -o /dev/null -H "Origin: $EVIL_ORIGIN" "$BASE_URL/health"
if grep -i '^Access-Control-Allow-Origin:' "$TMP/cors.headers" | grep -Fq "$EVIL_ORIGIN"; then
  fail "untrusted Origin was reflected by CORS"
fi
if grep -i '^Access-Control-Allow-Origin:' "$TMP/cors.headers" | grep -Fq '*'; then
  fail "wildcard CORS origin is enabled"
fi
pass "CORS rejects untrusted origins"

# XSS: React source must not bypass escaping with dangerouslySetInnerHTML.
if grep -R --include='*.tsx' --include='*.ts' -n 'dangerouslySetInnerHTML' "$FRONTEND_SRC" >/dev/null 2>&1; then
  fail "dangerouslySetInnerHTML is present in frontend source"
fi
pass "frontend does not bypass React escaping"

# Logout invalidates the refresh token. Reusing the logged-out refresh token must fail.
REFRESH_BEFORE_LOGOUT=$(awk '$6=="refresh_token" {print $7}' "$TMP/cookies2" | tail -1)
status=$(curl -sS -b "$TMP/cookies2" -o "$TMP/logout.json" -w '%{http_code}' -X POST \
  -H "Authorization: Bearer $ACCESS_TOKEN" -H "X-CSRF-Token: $CSRF" "$BASE_URL/api/auth/logout")
[ "$status" = "204" ] || fail "logout returned HTTP $status"
printf '# Netscape HTTP Cookie File\nlocalhost\tFALSE\t/api/auth\tFALSE\t2147483647\trefresh_token\t%s\nlocalhost\tFALSE\t/\tFALSE\t2147483647\tcsrf_token\t%s\n' "$REFRESH_BEFORE_LOGOUT" "$CSRF" > "$TMP/loggedout.cookies"
status=$(curl -sS -b "$TMP/loggedout.cookies" -o "$TMP/postlogout.json" -w '%{http_code}' -X POST \
  -H "X-CSRF-Token: $CSRF" "$BASE_URL/api/auth/refresh")
[ "$status" = "401" ] || fail "refresh token remained usable after logout; HTTP $status"
pass "logout invalidates refresh token"

# Rate limit: repeated invalid logins must eventually return 429.
rate_limited=0
i=1
while [ "$i" -le 25 ]; do
  status=$(curl -sS -o /dev/null -w '%{http_code}' -H 'Content-Type: application/json' \
    -d "{\"email\":\"rate-limit-security@example.test\",\"password\":\"wrong-password\"}" \
    "$BASE_URL/api/auth/login")
  if [ "$status" = "429" ]; then rate_limited=1; break; fi
  i=$((i + 1))
done
[ "$rate_limited" = "1" ] || fail "login rate limit did not return HTTP 429"
pass "login rate limit is enforced"

echo "Security checks passed"
