#!/usr/bin/env bash
# Builds Tinkerie for Android: the Go program for each Android ABI (with the
# NDK, so it uses Android's own DNS and libc), then the APK around it.
#   android/build.sh            release APK in dist/packages/Tinkerie-<ver>-android.apk
#   android/build.sh debug x86_64   debug APK with only the given ABIs (for emulators)
# Needs Go, the Android SDK (ANDROID_HOME), the NDK (ANDROID_NDK_LATEST_HOME or
# ANDROID_NDK_HOME), Java 17 and Gradle. Created by nrmluvdev for Tinkerie.
set -euo pipefail
cd "$(dirname "$0")/.."
kind=${1:-release}
shift || true
abis=${*:-arm64-v8a armeabi-v7a x86_64 x86}
v=$(sed -n 's/^VERSION=//p' build.sh)
# 0.13.0-alpha -> 1300: grows with every version
IFS=. read -r ma mi pa <<<"${v%%-*}"
code=$((ma * 10000 + mi * 100 + pa))
ndk=${ANDROID_NDK_LATEST_HOME:-${ANDROID_NDK_HOME:-}}
[ -d "$ndk" ] || { echo "No Android NDK found (set ANDROID_NDK_HOME)"; exit 1; }
tc="$ndk/toolchains/llvm/prebuilt/linux-x86_64/bin"
api=21 # Android 5.0
rm -rf android/app/src/main/jniLibs
for abi in $abis; do
  case $abi in
    arm64-v8a)   ga=arm64; cc=aarch64-linux-android$api-clang ;;
    armeabi-v7a) ga=arm;   cc=armv7a-linux-androideabi$api-clang ;;
    x86_64)      ga=amd64; cc=x86_64-linux-android$api-clang ;;
    x86)         ga=386;   cc=i686-linux-android$api-clang ;;
    *) echo "unknown ABI $abi"; exit 1 ;;
  esac
  out=android/app/src/main/jniLibs/$abi/libtinkerie.so
  mkdir -p "$(dirname "$out")"
  echo "Go for $abi"
  # 16 KB pages: newer phones need the program laid out for them
  CGO_ENABLED=1 GOOS=android GOARCH=$ga GOARM=7 CC="$tc/$cc" \
    go build -trimpath -ldflags "-s -w -extldflags=-Wl,-z,max-page-size=16384" -o "$out" .
done
task=assembleRelease; [ "$kind" = debug ] && task=assembleDebug
(cd android && TINKERIE_VERSION=$v TINKERIE_VERSION_CODE=$code gradle --no-daemon -q $task)
mkdir -p dist/packages
if [ "$kind" = debug ]; then
  cp android/app/build/outputs/apk/debug/app-debug.apk dist/packages/Tinkerie-$v-android-debug.apk
else
  cp android/app/build/outputs/apk/release/app-release.apk dist/packages/Tinkerie-$v-android.apk
fi
ls -la dist/packages/*android*
