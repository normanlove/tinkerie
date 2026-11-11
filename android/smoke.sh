#!/usr/bin/env bash
# Emulator smoke test for Tinkerie on Android (run by .github/workflows/android.yml):
# installs the debug APK, starts it, checks the Go launcher answers, opens two
# apps through it, and keeps screenshots and logs in the given folder.
set -uo pipefail
apk=$1 out=$2
mkdir -p "$out"
fail() { echo "SMOKE FAIL: $*"; adb logcat -d > "$out/logcat.txt"; exit 1; }
adb install -r -g "$apk" || fail "install"
adb shell am start -W -n net.tinkerie.app/.MainActivity
for i in $(seq 1 60); do
  log=$(adb shell run-as net.tinkerie.app cat files/home/logs/launcher.log 2>/dev/null | tr -d '\r')
  url=$(echo "$log" | grep -o 'http://127.0.0.1:[0-9]*/' | tail -1)
  [ -n "$url" ] && break
  sleep 2
done
echo "$log" > "$out/launcher.log"
[ -n "$url" ] || fail "the Go launcher didn't start"
port=${url##*:}; port=${port%/}
adb forward tcp:18080 tcp:"$port"
curl -fsS localhost:18080/api/apps > "$out/apps.json" || fail "launcher API"
sleep 8
adb exec-out screencap -p > "$out/1-launcher.png"
n=2
for app in codie wirie; do
  curl -fsS -X POST -H 'Content-Type: application/json' -d "{\"App\":\"$app\"}" localhost:18080/api/launch || fail "launch $app"
  sleep 20
  adb exec-out screencap -p > "$out/$n-$app.png"; n=$((n + 1))
  adb shell run-as net.tinkerie.app cat files/home/logs/$app.log > "$out/$app.log" 2>/dev/null
done
adb shell dumpsys activity recents | grep -E "realActivity|intent=" | head -20 > "$out/recents.txt"
adb shell run-as net.tinkerie.app ls -laR files/home > "$out/files.txt" 2>&1
adb logcat -d > "$out/logcat.txt"
if grep -q "FATAL EXCEPTION" "$out/logcat.txt"; then fail "the app crashed (see logcat.txt)"; fi
grep -q "AppActivity" "$out/recents.txt" || echo "warning: no app window in Recents"
echo "SMOKE OK"
