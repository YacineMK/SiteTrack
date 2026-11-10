#!/usr/bin/env bash
# End-to-end smoke test against a running API at :8080.
# Usage: ./tmp/e2e.sh   (expects a fresh DB; unique emails per run)
set -euo pipefail

BASE=${BASE:-http://localhost:8080}
RUN=$(date +%s)
PASS=0; FAIL=0

say()  { printf '\n\033[1m%s\033[0m\n' "$*"; }
ok()   { PASS=$((PASS+1)); printf '  ok   %s\n' "$*"; }
bad()  { FAIL=$((FAIL+1)); printf '  FAIL %s\n' "$*"; }

# check <name> <expected-status> <actual-status>
check() {
  if [ "$2" = "$3" ]; then ok "$1 ($3)"; else bad "$1: want $2, got $3"; fi
}

# req <jar> <method> <path> [body] -> prints "status body"
req() {
  local jar=$1 method=$2 path=$3 body=${4:-}
  case "$path" in /auth/*) sleep 2.1 ;; esac   # auth limiter: 5 per 10s
  if [ -n "$body" ]; then
    curl -s -b "$jar" -c "$jar" -X "$method" "$BASE$path" -H 'Content-Type: application/json' -d "$body" -w '\n%{http_code}'
  else
    curl -s -b "$jar" -c "$jar" -X "$method" "$BASE$path" -w '\n%{http_code}'
  fi
}
body_of() { sed '$d'; }
code_of() { tail -n1; }

# reqj returns body only, failing loudly on non-2xx
reqj() {
  local out code
  out=$(req "$@"); code=$(printf '%s' "$out" | code_of)
  case "$code" in 2*) printf '%s' "$out" | body_of ;; *) echo "HTTP $code: $(printf '%s' "$out" | body_of)" >&2; return 1 ;; esac
}

say "health"
check "GET /health" 200 "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/health")"

ADMIN=/tmp/e2e-admin.jar; MGR=/tmp/e2e-mgr.jar; USR=/tmp/e2e-usr.jar
rm -f "$ADMIN" "$MGR" "$USR"

say "register"
A_EMAIL="admin$RUN@example.com"; M_EMAIL="mgr$RUN@example.com"; U_EMAIL="usr$RUN@example.com"
check "register super_admin" 201 "$(req "$ADMIN" POST /auth/register "{\"name\":\"Admin\",\"email\":\"$A_EMAIL\",\"password\":\"Password123!\",\"app_role\":\"super_admin\"}" | code_of)"
check "register manager"     201 "$(req "$MGR"   POST /auth/register "{\"name\":\"Mgr\",\"email\":\"$M_EMAIL\",\"password\":\"Password123!\",\"app_role\":\"company_store_manager\"}" | code_of)"
check "register user"        201 "$(req "$USR"   POST /auth/register "{\"name\":\"Usr\",\"email\":\"$U_EMAIL\",\"password\":\"Password123!\",\"app_role\":\"user\"}" | code_of)"
check "register duplicate"   409 "$(req "$ADMIN" POST /auth/register "{\"name\":\"Admin\",\"email\":\"$A_EMAIL\",\"password\":\"Password123!\",\"app_role\":\"super_admin\"}" | code_of)"
check "email-available false" 200 "$(req "$ADMIN" GET "/auth/email-available?email=$A_EMAIL" | code_of)"
AVAIL=$(req "$ADMIN" GET "/auth/email-available?email=fresh$RUN@example.com" | body_of | jq -r .available)
[ "$AVAIL" = "true" ] && ok "email-available true" || bad "email-available expected true"

say "login / me"
check "login admin"   200 "$(req "$ADMIN" POST /auth/login "{\"email\":\"$A_EMAIL\",\"password\":\"Password123!\"}" | code_of)"
check "login manager" 200 "$(req "$MGR"   POST /auth/login "{\"email\":\"$M_EMAIL\",\"password\":\"Password123!\"}" | code_of)"
check "login user"    200 "$(req "$USR"   POST /auth/login "{\"email\":\"$U_EMAIL\",\"password\":\"Password123!\"}" | code_of)"
check "login bad pw"  401 "$(req /tmp/none.jar POST /auth/login "{\"email\":\"$U_EMAIL\",\"password\":\"wrong\"}" | code_of)"
ME=$(reqj "$USR" GET /auth/me)
[ "$(printf '%s' "$ME" | jq -r .email)" = "$U_EMAIL" ] && ok "GET /auth/me" || bad "me email mismatch"

say "projects"
P=$(reqj "$USR" POST /projects '{"name":"E2E Project","description":"smoke","planned_start_date":"2026-01-01","planned_end_date":"2026-12-31"}')
PID=$(printf '%s' "$P" | jq -r .display_id)
[ "$(printf '%s' "$P" | jq -r .is_admin)" = "true" ] && ok "creator is admin" || bad "creator is_admin false"
LIST=$(reqj "$USR" GET "/projects?q=E2E&sort=name")
[ "$(printf '%s' "$LIST" | jq -r '.items[0].is_admin')" = "true" ] && ok "list is_admin true for creator" || bad "list is_admin wrong"
check "get project" 200 "$(req "$USR" GET "/projects/$PID" | code_of)"
check "patch project" 200 "$(req "$USR" PATCH "/projects/$PID" '{"name":"E2E Project v2","description":"smoke","planned_start_date":"2026-01-01","planned_end_date":"2026-12-31"}' | code_of)"
check "non-member 404" 404 "$(req "$MGR" GET "/projects/$PID" | code_of)"

say "members / invitations"
check "list members" 200 "$(req "$USR" GET "/projects/$PID/members" | code_of)"
INV=$(reqj "$USR" POST "/projects/$PID/invitations" "{\"email\":\"$M_EMAIL\",\"role\":\"member\"}")
IID=$(printf '%s' "$INV" | jq -r .id)
[ "$IID" != "null" ] && ok "invite created" || bad "invite id missing"
check "pending invite visible in manager notifications" 200 "$(req "$MGR" GET '/notifications?unread=true' | code_of)"
check "accept invite" 204 "$(req "$MGR" POST "/invitations/$IID/accept" | code_of)"
MEMBERS=$(reqj "$USR" GET "/projects/$PID/members")
[ "$(printf '%s' "$MEMBERS" | jq '[.items[]|select(.email=="'"$M_EMAIL"'")]|length')" = "1" ] && ok "manager joined project" || bad "manager not a member"
check "non-admin cannot invite" 403 "$(req "$MGR" POST "/projects/$PID/invitations" "{\"email\":\"$U_EMAIL\",\"role\":\"member\"}" | code_of)"

say "tasks"
T1=$(reqj "$USR" POST "/projects/$PID/tasks" '{"name":"A","planned_start_date":"2026-01-01","planned_end_date":"2026-02-01"}')
T2=$(reqj "$USR" POST "/projects/$PID/tasks" '{"name":"B","planned_start_date":"2026-02-02","planned_end_date":"2026-03-01"}')
TID1=$(printf '%s' "$T1" | jq -r .display_id); TID2=$(printf '%s' "$T2" | jq -r .display_id)
[ "$TID1" = "$PID-T-0001" ] && ok "task display id PRJ-T-0001" || bad "task id $TID1"
check "list tasks" 200 "$(req "$USR" GET "/projects/$PID/tasks?sort=name" | code_of)"
check "list tasks late filter" 200 "$(req "$USR" GET "/projects/$PID/tasks?late=true" | code_of)"
check "add dependency B->A" 204 "$(req "$USR" POST "/projects/$PID/tasks/$TID2/dependencies" "{\"depends_on\":\"$TID1\"}" | code_of)"
check "cycle A->B rejected" 409 "$(req "$USR" POST "/projects/$PID/tasks/$TID1/dependencies" "{\"depends_on\":\"$TID2\"}" | code_of)"
BLOCKED=$(reqj "$USR" GET "/projects/$PID/tasks/$TID2")
[ "$(printf '%s' "$BLOCKED" | jq -r .blocked)" = "true" ] && ok "B is blocked" || bad "B should be blocked"
check "start blocked B rejected" 409 "$(req "$USR" POST "/projects/$PID/tasks/$TID2/start" | code_of)"
check "start A" 200 "$(req "$USR" POST "/projects/$PID/tasks/$TID1/start" | code_of)"
check "done A" 200 "$(req "$USR" POST "/projects/$PID/tasks/$TID1/done" | code_of)"
check "start B now allowed" 200 "$(req "$USR" POST "/projects/$PID/tasks/$TID2/start" | code_of)"
check "reopen A blocked by started downstream" 409 "$(req "$USR" POST "/projects/$PID/tasks/$TID1/reopen" | code_of)"
DETAIL=$(reqj "$USR" GET "/projects/$PID/tasks/$TID2")
[ "$(printf '%s' "$DETAIL" | jq '.upstream|length')" = "1" ] && ok "detail upstream" || bad "detail upstream wrong"

say "materials"
MAT=$(reqj "$MGR" POST /materials "{\"name\":\"Cement $RUN\",\"category\":\"binder\",\"unit\":\"bag\"}")
MID=$(printf '%s' "$MAT" | jq -r .id)
[ "$MID" != "null" ] && ok "create material" || bad "material id missing"
check "list materials search" 200 "$(req "$MGR" GET "/materials?q=Cement" | code_of)"

say "company store"
check "add stock" 201 "$(req "$MGR" POST /company-store/stock/additions "{\"material_id\":\"$MID\",\"quantity\":10}" | code_of)"
check "set reorder" 200 "$(req "$MGR" PUT "/company-store/stock/$MID/reorder-point" '{"reorder_point":20}' | code_of)"
STOCK=$(reqj "$MGR" GET "/company-store/stock?q=Cement&status=low&sort=quantity")
[ "$(printf '%s' "$STOCK" | jq -r '.items[0].status')" = "low" ] && ok "status low (qty<=reorder)" || bad "status expected low"
check "filter out_of_stock" 200 "$(req "$MGR" GET '/company-store/stock?status=out_of_stock' | code_of)"
check "user cannot read company store" 403 "$(req "$USR" GET /company-store/stock | code_of)"

say "deliveries"
D=$(reqj "$MGR" POST /company-store/deliveries "{\"receiver_project\":\"$PID\",\"lines\":[{\"material_id\":\"$MID\",\"quantity\":4}]}")
DID=$(printf '%s' "$D" | jq -r .id)
[ "$(printf '%s' "$D" | jq -r .status)" = "in_transit" ] && ok "delivery in transit" || bad "delivery status wrong"
[ "$(printf '%s' "$D" | jq '.lines|length')" = "1" ] && ok "delivery returns lines" || bad "delivery lines missing"
LST=$(reqj "$MGR" GET "/company-store/deliveries?direction=out&status=in_transit")
[ "$(printf '%s' "$LST" | jq '.items|length')" -ge 1 ] && ok "company delivery list + filter" || bad "company delivery list empty"
[ "$(printf '%s' "$LST" | jq -r '.items[0].receiver_display_id')" = "$PID" ] && ok "list has project display id" || bad "receiver_display_id wrong"
PS=$(reqj "$USR" GET "/projects/$PID/deliveries?direction=in")
[ "$(printf '%s' "$PS" | jq '.items|length')" -ge 1 ] && ok "project delivery list" || bad "project delivery list empty"
check "record delivery (project receiver)" 200 "$(req "$USR" POST "/deliveries/$DID/record" | code_of)"
PSTOCK=$(reqj "$USR" GET "/projects/$PID/store/items?q=Cement")
[ "$(printf '%s' "$PSTOCK" | jq -r '.items[0].quantity')" = "4" ] && ok "project stock credited" || bad "project stock expected 4"
check "record twice rejected" 409 "$(req "$USR" POST "/deliveries/$DID/record" | code_of)"
check "list project store sort" 200 "$(req "$USR" GET "/projects/$PID/store/items?sort=last_delivery" | code_of)"

say "notifications"
check "unread count" 200 "$(req "$USR" GET /notifications/unread-count | code_of)"
check "list notifications" 200 "$(req "$USR" GET /notifications | code_of)"

say "logout"
check "logout" 204 "$(req "$USR" POST /auth/logout | code_of)"
check "me after logout 401" 401 "$(req "$USR" GET /auth/me | code_of)"

say "cleanup"
check "delete project with deliveries rejected" 409 "$(req "$ADMIN" DELETE "/projects/$PID" | code_of)"
EMPTY=$(reqj "$ADMIN" POST /projects '{"name":"E2E Empty","description":"","planned_start_date":"2026-01-01","planned_end_date":"2026-02-01"}')
EPID=$(printf '%s' "$EMPTY" | jq -r .display_id)
check "delete empty project" 204 "$(req "$ADMIN" DELETE "/projects/$EPID" | code_of)"

printf '\n\033[1m%d passed, %d failed\033[0m\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
