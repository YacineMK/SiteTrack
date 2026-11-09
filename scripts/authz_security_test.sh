#!/usr/bin/env bash
set -u
ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT"

BASE=http://localhost:8080
PASS=0
FAIL=0
FAILURES=()

ok() {
  if [ "$2" = "$3" ]; then
    PASS=$((PASS + 1))
    echo "PASS $1 ($3)"
  else
    FAIL=$((FAIL + 1))
    FAILURES+=("$1: want $2 got $3")
    echo "FAIL $1 (want $2 got $3)"
  fi
}

psql() {
  docker exec sitetrack-postgres-1 psql -U sitetrack -d sitetrack -qAtc "$1" 2>/dev/null
}

redis() {
  docker exec sitetrack-redis-1 redis-cli -a sitetrack --no-auth-warning "$@" 2>/dev/null
}

cleanup() {
  [ -n "${APP_PID:-}" ] && kill "$APP_PID" 2>/dev/null
  [ -n "${STUB_PID:-}" ] && kill "$STUB_PID" 2>/dev/null
  lsof -ti:8080 2>/dev/null | xargs -r kill 2>/dev/null
  lsof -ti:9099 2>/dev/null | xargs -r kill 2>/dev/null
}
trap cleanup EXIT

echo "== migrate =="
make migrate >/dev/null 2>&1 || { echo "migration failed"; exit 1; }

cat > /tmp/authz_resend_stub.py <<'PY'
from http.server import BaseHTTPRequestHandler, HTTPServer
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', '0'))
        body = self.rfile.read(n)
        with open('/tmp/authz_resend.log', 'ab') as f:
            f.write(body + b'\n')
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(b'{"id":"stub"}')
    def log_message(self, *a):
        pass
HTTPServer(('127.0.0.1', 9099), H).serve_forever()
PY
rm -f /tmp/authz_resend.log
python3 /tmp/authz_resend_stub.py &
STUB_PID=$!

lsof -ti:8080 2>/dev/null | xargs -r kill 2>/dev/null
sleep 1
RESEND_BASE_URL=http://127.0.0.1:9099 OUTBOX_POLL_INTERVAL=1s nohup go run ./cmd >/tmp/authz_app.log 2>&1 &
APP_PID=$!

for _ in $(seq 1 30); do
  if curl -s -o /dev/null "$BASE/health"; then break; fi
  sleep 1
done
if ! grep -q listening /tmp/authz_app.log; then
  echo "server did not start"; tail -20 /tmp/authz_app.log; exit 1
fi

TS=$(date +%s%N)
SOURCE_SUPER=127.0.0.5
SOURCE_MGR=127.0.0.6
SOURCE_A=127.0.0.2
SOURCE_B=127.0.0.3
SOURCE_C=127.0.0.4

JAR_SUPER=$(mktemp); JAR_MGR=$(mktemp); JAR_A=$(mktemp); JAR_B=$(mktemp); JAR_C=$(mktemp)
EMAIL_SUPER="super$TS@example.com"
EMAIL_MGR="mgr$TS@example.com"
EMAIL_A="owner$TS@example.com"
EMAIL_B="member$TS@example.com"
EMAIL_C="outsider$TS@example.com"
PASSWORD="Password123!"

as() { local src=$1; shift; curl --interface "$src" -s "$@"; }
code_as() { local src=$1; shift; curl --interface "$src" -s -o /tmp/authz_body.json -w "%{http_code}" "$@"; }

reg() {
  local src=$1 email=$2 role=$3 jar=$4
  code_as "$src" -c "$jar" -X POST "$BASE/auth/register" -H 'Content-Type: application/json' \
    -d "{\"name\":\"u\",\"email\":\"$email\",\"password\":\"$PASSWORD\",\"app_role\":\"$role\"}"
}
login() {
  local src=$1 email=$2 jar=$3
  code_as "$src" -c "$jar" -X POST "$BASE/auth/login" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$email\",\"password\":\"$PASSWORD\"}"
}

echo "== authentication =="
ok "no cookie projects" 401 "$(code_as $SOURCE_C "$BASE/projects")"
ok "no cookie company store" 401 "$(code_as $SOURCE_C "$BASE/company-store/stock")"
ok "no cookie resources" 401 "$(code_as $SOURCE_C "$BASE/resources")"
ok "invalid login" 401 "$(code_as $SOURCE_C -X POST "$BASE/auth/login" -H 'Content-Type: application/json' -d '{"email":"nobody@example.com","password":"wrongpass"}')"
ok "register super" 201 "$(reg $SOURCE_SUPER "$EMAIL_SUPER" super_admin "$JAR_SUPER")"
ok "register manager" 201 "$(reg $SOURCE_MGR "$EMAIL_MGR" company_store_manager "$JAR_MGR")"
ok "register A" 201 "$(reg $SOURCE_A "$EMAIL_A" user "$JAR_A")"
ok "register B" 201 "$(reg $SOURCE_B "$EMAIL_B" user "$JAR_B")"
ok "register C" 201 "$(reg $SOURCE_C "$EMAIL_C" user "$JAR_C")"
ok "reject unknown role" 400 "$(reg $SOURCE_C "bad$TS@example.com" hacker "$(mktemp)")"

for pair in "$SOURCE_SUPER:$EMAIL_SUPER:$JAR_SUPER" "$SOURCE_MGR:$EMAIL_MGR:$JAR_MGR" "$SOURCE_A:$EMAIL_A:$JAR_A" "$SOURCE_B:$EMAIL_B:$JAR_B" "$SOURCE_C:$EMAIL_C:$JAR_C"; do
  src=${pair%%:*}; rest=${pair#*:}; email=${rest%%:*}; jar=${rest#*:}
  ok "login $email" 200 "$(login "$src" "$email" "$jar")"
done

echo "== RBAC policies =="
ok "user company store" 403 "$(code_as $SOURCE_A -b $JAR_A "$BASE/company-store/stock")"
ok "manager company store" 200 "$(code_as $SOURCE_MGR -b $JAR_MGR "$BASE/company-store/stock")"
ok "super company store" 200 "$(code_as $SOURCE_SUPER -b $JAR_SUPER "$BASE/company-store/stock")"
ok "user resources" 403 "$(code_as $SOURCE_A -b $JAR_A "$BASE/resources")"
ok "super resources" 200 "$(code_as $SOURCE_SUPER -b $JAR_SUPER "$BASE/resources")"
ok "user roles" 403 "$(code_as $SOURCE_A -b $JAR_A "$BASE/roles")"
ok "super roles" 200 "$(code_as $SOURCE_SUPER -b $JAR_SUPER "$BASE/roles")"
ok "system roles flagged" true "$(jq -r '.[]|select(.name=="user")|.is_system' /tmp/authz_body.json)"

RNAME="clerk$TS"
ok "create custom role" 201 "$(code_as $SOURCE_SUPER -b $JAR_SUPER -X POST "$BASE/roles" -H 'Content-Type: application/json' -d "{\"name\":\"$RNAME\",\"description\":\"store reader\"}")"
RID=$(jq -r .id /tmp/authz_body.json)
ok "invalid policy resource" 400 "$(code_as $SOURCE_SUPER -b $JAR_SUPER -X POST "$BASE/roles/$RID/policies" -H 'Content-Type: application/json' -d '{"resource":"nope","action":"read"}')"
ok "invalid policy action" 400 "$(code_as $SOURCE_SUPER -b $JAR_SUPER -X POST "$BASE/roles/$RID/policies" -H 'Content-Type: application/json' -d '{"resource":"company_store","action":"delete"}')"
ok "add read policy" 201 "$(code_as $SOURCE_SUPER -b $JAR_SUPER -X POST "$BASE/roles/$RID/policies" -H 'Content-Type: application/json' -d '{"resource":"company_store","action":"read"}')"

AID=$(psql "SELECT id FROM users WHERE email='$EMAIL_A';" | head -1)
ok "user cannot assign roles" 403 "$(code_as $SOURCE_A -b $JAR_A -X PATCH "$BASE/users/$AID/role" -H 'Content-Type: application/json' -d "{\"role\":\"$RNAME\"}")"
ok "assign unknown role" 400 "$(code_as $SOURCE_SUPER -b $JAR_SUPER -X PATCH "$BASE/users/$AID/role" -H 'Content-Type: application/json' -d '{"role":"missing"}')"
ok "assign custom role" 200 "$(code_as $SOURCE_SUPER -b $JAR_SUPER -X PATCH "$BASE/users/$AID/role" -H 'Content-Type: application/json' -d "{\"role\":\"$RNAME\"}")"
ok "old token revoked" 401 "$(code_as $SOURCE_A -b $JAR_A "$BASE/auth/me")"
ok "re-login A" 200 "$(login $SOURCE_A "$EMAIL_A" "$JAR_A")"
ok "custom role read store" 200 "$(code_as $SOURCE_A -b $JAR_A "$BASE/company-store/stock")"
ok "custom role write denied" 403 "$(code_as $SOURCE_A -b $JAR_A -X POST "$BASE/company-store/stock/additions" -H 'Content-Type: application/json' -d '{"material_id":"00000000-0000-0000-0000-000000000000","quantity":1}')"
ok "restore A role" 200 "$(code_as $SOURCE_SUPER -b $JAR_SUPER -X PATCH "$BASE/users/$AID/role" -H 'Content-Type: application/json' -d '{"role":"user"}')"
ok "re-login A2" 200 "$(login $SOURCE_A "$EMAIL_A" "$JAR_A")"

echo "== project membership =="
PNAME="Secure$TS"
ok "A creates project" 201 "$(code_as $SOURCE_A -b $JAR_A -X POST "$BASE/projects" -H 'Content-Type: application/json' -d "{\"name\":\"$PNAME\",\"description\":\"x\",\"planned_start_date\":\"2026-11-01\",\"planned_end_date\":\"2026-12-31\"}")"
P=$(jq -r .display_id /tmp/authz_body.json)
ok "creator is admin" true "$(jq -r .is_admin /tmp/authz_body.json)"
ok "B cannot see project" 404 "$(code_as $SOURCE_B -b $JAR_B "$BASE/projects/$P")"
ok "B cannot list tasks" 404 "$(code_as $SOURCE_B -b $JAR_B "$BASE/projects/$P/tasks")"
ok "B cannot see store" 404 "$(code_as $SOURCE_B -b $JAR_B "$BASE/projects/$P/store/items")"
ok "B cannot see deliveries" 404 "$(code_as $SOURCE_B -b $JAR_B "$BASE/projects/$P/deliveries")"
ok "C guessed project" 404 "$(code_as $SOURCE_C -b $JAR_C "$BASE/projects/$P")"
ok "super sees project" 200 "$(code_as $SOURCE_SUPER -b $JAR_SUPER "$BASE/projects/$P")"

echo "== invitations =="
ok "B invites before membership denied" 404 "$(code_as $SOURCE_B -b $JAR_B -X POST "$BASE/projects/$P/invitations" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL_C\",\"role\":\"member\"}")"
ok "A invites B" 201 "$(code_as $SOURCE_A -b $JAR_A -X POST "$BASE/projects/$P/invitations" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL_B\",\"role\":\"member\"}")"
INV_B=$(jq -r .id /tmp/authz_body.json)
ok "duplicate invite" 409 "$(code_as $SOURCE_A -b $JAR_A -X POST "$BASE/projects/$P/invitations" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL_B\",\"role\":\"member\"}")"
ok "invite unknown user" 404 "$(code_as $SOURCE_A -b $JAR_A -X POST "$BASE/projects/$P/invitations" -H 'Content-Type: application/json' -d '{"email":"ghost@example.com","role":"member"}')"
ok "invite invalid role" 400 "$(code_as $SOURCE_A -b $JAR_A -X POST "$BASE/projects/$P/invitations" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL_C\",\"role\":\"boss\"}")"
ok "redis invite key" 1 "$(redis EXISTS "invite:$INV_B")"
ok "redis invite ttl <= 1h" 1 "$( [ "$(redis TTL "invite:$INV_B")" -le 3600 ] && echo 1 || echo 0 )"
ok "C cannot accept B invite" 404 "$(code_as $SOURCE_C -b $JAR_C -X POST "$BASE/invitations/$INV_B/accept")"
ok "B unread count" 1 "$( [ "$(code_as $SOURCE_B -b $JAR_B "$BASE/notifications/unread-count" >/dev/null; jq -r .unread /tmp/authz_body.json)" -ge 1 ] && echo 1 || echo 0 )"
ok "B accepts" 204 "$(code_as $SOURCE_B -b $JAR_B -X POST "$BASE/invitations/$INV_B/accept")"
ok "B membership created" 1 "$(psql "SELECT count(*) FROM project_members WHERE project_id=(SELECT id FROM projects WHERE display_id='$P') AND user_id=(SELECT id FROM users WHERE email='$EMAIL_B');")"
ok "B member can view" 200 "$(code_as $SOURCE_B -b $JAR_B "$BASE/projects/$P")"
ok "invite key deleted" 0 "$(redis EXISTS "invite:$INV_B")"
ok "double accept" 404 "$(code_as $SOURCE_B -b $JAR_B -X POST "$BASE/invitations/$INV_B/accept")"
ok "invite member again" 409 "$(code_as $SOURCE_A -b $JAR_A -X POST "$BASE/projects/$P/invitations" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL_B\",\"role\":\"member\"}")"

ok "A invites C" 201 "$(code_as $SOURCE_A -b $JAR_A -X POST "$BASE/projects/$P/invitations" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL_C\",\"role\":\"member\"}")"
INV_C=$(jq -r .id /tmp/authz_body.json)
ok "C rejects" 204 "$(code_as $SOURCE_C -b $JAR_C -X POST "$BASE/invitations/$INV_C/reject")"
ok "rejected not member" 0 "$(psql "SELECT count(*) FROM project_members WHERE project_id=(SELECT id FROM projects WHERE display_id='$P') AND user_id=(SELECT id FROM users WHERE email='$EMAIL_C');")"

ok "A invites C again" 201 "$(code_as $SOURCE_A -b $JAR_A -X POST "$BASE/projects/$P/invitations" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL_C\",\"role\":\"member\"}")"
INV_C2=$(jq -r .id /tmp/authz_body.json)
ok "A withdraws" 204 "$(code_as $SOURCE_A -b $JAR_A -X DELETE "$BASE/projects/$P/invitations/$INV_C2")"
ok "withdrawn accept" 404 "$(code_as $SOURCE_C -b $JAR_C -X POST "$BASE/invitations/$INV_C2/accept")"

ok "A invites C third" 201 "$(code_as $SOURCE_A -b $JAR_A -X POST "$BASE/projects/$P/invitations" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL_C\",\"role\":\"member\"}")"
INV_C3=$(jq -r .id /tmp/authz_body.json)
redis DEL "invite:$INV_C3" >/dev/null
ok "expired accept" 410 "$(code_as $SOURCE_C -b $JAR_C -X POST "$BASE/invitations/$INV_C3/accept")"

ok "member B cannot edit project" 403 "$(code_as $SOURCE_B -b $JAR_B -X PATCH "$BASE/projects/$P" -H 'Content-Type: application/json' -d "{\"name\":\"$PNAME\",\"description\":\"x\",\"planned_start_date\":\"2026-11-01\",\"planned_end_date\":\"2026-12-31\"}")"
BID=$(psql "SELECT id FROM users WHERE email='$EMAIL_B';" | head -1)
ok "B cannot change members" 403 "$(code_as $SOURCE_B -b $JAR_B -X PATCH "$BASE/projects/$P/members/$AID" -H 'Content-Type: application/json' -d '{"is_admin":false}')"
ok "A promotes B" 204 "$(code_as $SOURCE_A -b $JAR_A -X PATCH "$BASE/projects/$P/members/$BID" -H 'Content-Type: application/json' -d '{"is_admin":true}')"
ok "B admin can edit" 200 "$(code_as $SOURCE_B -b $JAR_B -X PATCH "$BASE/projects/$P" -H 'Content-Type: application/json' -d "{\"name\":\"$PNAME v2\",\"description\":\"x\",\"planned_start_date\":\"2026-11-01\",\"planned_end_date\":\"2026-12-31\"}")"
ok "B demotes A" 204 "$(code_as $SOURCE_B -b $JAR_B -X PATCH "$BASE/projects/$P/members/$AID" -H 'Content-Type: application/json' -d '{"is_admin":false}')"
ok "last admin demote" 409 "$(code_as $SOURCE_B -b $JAR_B -X PATCH "$BASE/projects/$P/members/$BID" -H 'Content-Type: application/json' -d '{"is_admin":false}')"
ok "last admin remove" 409 "$(code_as $SOURCE_B -b $JAR_B -X DELETE "$BASE/projects/$P/members/$BID")"

echo "== delivery authorization =="
ok "manager creates material" 201 "$(code_as $SOURCE_MGR -b $JAR_MGR -X POST "$BASE/materials" -H 'Content-Type: application/json' -d "{\"name\":\"Mat$TS\",\"category\":\"x\",\"unit\":\"kg\"}")"
MID=$(jq -r .id /tmp/authz_body.json)
ok "manager adds stock" 201 "$(code_as $SOURCE_MGR -b $JAR_MGR -X POST "$BASE/company-store/stock/additions" -H 'Content-Type: application/json' -d "{\"material_id\":\"$MID\",\"quantity\":100}")"
ok "user cannot list company deliveries" 403 "$(code_as $SOURCE_A -b $JAR_A "$BASE/company-store/deliveries")"
ok "manager creates company delivery" 201 "$(code_as $SOURCE_MGR -b $JAR_MGR -X POST "$BASE/company-store/deliveries" -H 'Content-Type: application/json' -d "{\"receiver_project\":\"$P\",\"lines\":[{\"material_id\":\"$MID\",\"quantity\":5}]}")"
D1=$(jq -r .id /tmp/authz_body.json)
ok "sent_by recorded" 1 "$( [ "$(jq -r .sent_by /tmp/authz_body.json)" != "null" ] && echo 1 || echo 0 )"
ok "outsider cannot read delivery" 404 "$(code_as $SOURCE_C -b $JAR_C "$BASE/deliveries/$D1")"
ok "receiver B records" 200 "$(code_as $SOURCE_B -b $JAR_B -X POST "$BASE/deliveries/$D1/record")"
ok "received_by recorded" 1 "$( [ "$(jq -r .received_by /tmp/authz_body.json)" != "null" ] && echo 1 || echo 0 )"
ok "receiver cannot cancel" 404 "$(code_as $SOURCE_B -b $JAR_B -X POST "$BASE/deliveries/$D1/cancel")"

ok "manager sends second" 201 "$(code_as $SOURCE_MGR -b $JAR_MGR -X POST "$BASE/company-store/deliveries" -H 'Content-Type: application/json' -d "{\"receiver_project\":\"$P\",\"lines\":[{\"material_id\":\"$MID\",\"quantity\":4}]}")"
D2=$(jq -r .id /tmp/authz_body.json)
ok "receiver cannot cancel" 404 "$(code_as $SOURCE_B -b $JAR_B -X POST "$BASE/deliveries/$D2/cancel")"
ok "sender manager cancels" 200 "$(code_as $SOURCE_MGR -b $JAR_MGR -X POST "$BASE/deliveries/$D2/cancel")"
ok "cancelled cannot record" 409 "$(code_as $SOURCE_B -b $JAR_B -X POST "$BASE/deliveries/$D2/record")"

ok "B sends project delivery" 201 "$(code_as $SOURCE_B -b $JAR_B -X POST "$BASE/projects/$P/deliveries" -H 'Content-Type: application/json' -d "{\"lines\":[{\"material_id\":\"$MID\",\"quantity\":2}]}")"
D3=$(jq -r .id /tmp/authz_body.json)
ok "sender cannot record own" 404 "$(code_as $SOURCE_B -b $JAR_B -X POST "$BASE/deliveries/$D3/record")"
ok "company receiver records" 200 "$(code_as $SOURCE_MGR -b $JAR_MGR -X POST "$BASE/deliveries/$D3/record")"

echo "== rate limiting =="
CODES=""
for _ in $(seq 1 6); do
  CODES="$CODES $(curl --interface 127.0.0.9 -s -o /dev/null -w '%{http_code}' -X POST "$BASE/auth/login" -H 'Content-Type: application/json' -d '{"email":"nobody@example.com","password":"wrongpass"}')"
done
if echo "$CODES" | grep -q 429; then PASS=$((PASS+1)); echo "PASS auth rate limit (429)"; else FAIL=$((FAIL+1)); FAILURES+=("auth rate limit missing"); echo "FAIL auth rate limit"; fi
CODES=""
for _ in $(seq 1 61); do
  CODES="$CODES $(curl --interface 127.0.0.10 -s -o /dev/null -w '%{http_code}' "$BASE/health")"
done
if echo "$CODES" | grep -q 429; then PASS=$((PASS+1)); echo "PASS global rate limit (429)"; else FAIL=$((FAIL+1)); FAILURES+=("global rate limit missing"); echo "FAIL global rate limit"; fi

echo
echo "RESULT: $PASS passed, $FAIL failed"
if [ "$FAIL" -ne 0 ]; then
  printf ' - %s\n' "${FAILURES[@]}"
  echo "AUTHN/AUTHZ: FAIL — not production ready"
  exit 1
fi
echo "AUTHN/AUTHZ: PASS — no unauthorized access observed in this suite"
echo "Note: this is a behavioral check, not a formal audit; SSE/rate-limit state is per-instance."
