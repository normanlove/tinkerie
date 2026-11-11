#!/bin/sh
# Builds Tinkerie. Needs Go 1.24+ (standard library only). Everything is plain
# Go except the Mac build, which uses Apple's IOKit (USB scopes, logic
# analysers, meter cables) through cgo: build it on a Mac with Xcode's
# command-line tools. Built elsewhere, the Mac app works without USB devices.
#
#   ./build.sh            this computer's system (Windows exe, Mac app, Linux or Pi program)
#   ./build.sh windows    Tinkerie.exe
#   ./build.sh mac        dist/mac-apple-silicon/Tinkerie.app
#   ./build.sh mac-intel  dist/mac-intel/Tinkerie.app
#   ./build.sh linux      dist/linux-x86_64/tinkerie
#   ./build.sh pi         dist/pi-64bit/tinkerie   (Raspberry Pi OS 64-bit: Pi 3, 4, 5)
#   ./build.sh pi32       dist/pi-32bit/tinkerie   (Raspberry Pi OS 32-bit)
#   ./build.sh all        every one of the above
#
# To change the icon: go run ./tools/mkicon, then (for Windows)
#   go run ./tools/mksyso -ico build/tinkerie.ico -out rsrc_windows_amd64.syso -version 0.22.0-alpha
set -e
cd "$(dirname "$0")"
VERSION=0.22.0-alpha
NUMVER=${VERSION%%-*} # macOS wants plain numbers
export CGO_ENABLED=0

unix() { # name os arch [arm]
  mkdir -p "dist/$1"
  GOOS=$2 GOARCH=$3 GOARM=$4 go build -trimpath -ldflags "-s -w" -o "dist/$1/tinkerie" .
  cp build/tinkerie.png build/install.sh build/install-usb.sh "dist/$1/"
  chmod +x "dist/$1/tinkerie" "dist/$1/install.sh" "dist/$1/install-usb.sh"
  echo "Built dist/$1/tinkerie"
}

mac() { # name arch
  app="dist/$1/Tinkerie.app/Contents"
  mkdir -p "$app/MacOS" "$app/Resources"
  if [ "$(uname -s)" = Darwin ]; then
    # IOKit through cgo; runs on macOS 11 and newer
    carch=$2; [ "$carch" = amd64 ] && carch=x86_64
    CGO_ENABLED=1 CC="clang -arch $carch" MACOSX_DEPLOYMENT_TARGET=11.0 CGO_CFLAGS="-mmacosx-version-min=11.0" CGO_LDFLAGS="-mmacosx-version-min=11.0" \
      GOOS=darwin GOARCH=$2 go build -trimpath -ldflags "-s -w" -o "$app/MacOS/tinkerie" .
  else
    echo "Note: building the Mac app without IOKit (no USB devices). Build on a Mac for the full app."
    GOOS=darwin GOARCH=$2 go build -trimpath -ldflags "-s -w" -o "$app/MacOS/tinkerie" .
  fi
  sed "s/VERSION/$NUMVER/g" build/Info.plist > "$app/Info.plist"
  cp build/tinkerie.icns "$app/Resources/"
  chmod +x "$app/MacOS/tinkerie"
  echo "Built dist/$1/Tinkerie.app"
}

windows() {
  GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui -s -w" -o Tinkerie.exe .
  echo "Built Tinkerie.exe"
}

target="$1"
if [ -z "$target" ]; then
  case "$(uname -s)-$(uname -m)" in
    Darwin-arm64) target=mac ;;
    Darwin-*) target=mac-intel ;;
    Linux-x86_64) target=linux ;;
    Linux-aarch64 | Linux-arm64) target=pi ;;
    Linux-armv7l | Linux-armv6l) target=pi32 ;;
    *) target=windows ;;
  esac
fi

case "$target" in
  windows) windows ;;
  mac) mac mac-apple-silicon arm64 ;;
  mac-intel) mac mac-intel amd64 ;;
  linux) unix linux-x86_64 linux amd64 ;;
  pi) unix pi-64bit linux arm64 ;;
  pi32) unix pi-32bit linux arm 7 ;;
  all)
    windows
    if [ -z "$SKIP_MAC" ]; then # the release builds the Mac apps on a Mac
      mac mac-apple-silicon arm64
      mac mac-intel amd64
    fi
    unix linux-x86_64 linux amd64
    unix pi-64bit linux arm64
    unix pi-32bit linux arm 7
    ;;
  *) echo "Unknown target $target. Use windows, mac, mac-intel, linux, pi, pi32 or all."; exit 2 ;;
esac
