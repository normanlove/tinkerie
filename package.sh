#!/bin/sh
# Builds every platform and packs the release files into dist/packages:
# a zip (Windows, Mac) or tar.gz (Linux, Pi) per platform, each with the
# program, its READ ME FIRST.txt (from build/readme) and the full source.
# Also copies Tinkerie.exe there on its own. Made by nrmluvdev.
set -e
cd "$(dirname "$0")"
VERSION=$(sed -n 's/^VERSION=//p' build.sh)
./build.sh all # with SKIP_MAC=1 the Mac apps already in dist/ are used
out=dist/packages
rm -rf "$out"
mkdir -p "$out"
src=$(mktemp -d)
git ls-files -z | xargs -0 tar cf - | tar xf - -C "$src"

pack() { # platform
  d="$out/Tinkerie-$VERSION-$1"
  mkdir -p "$d/source"
  cp -a "$src/." "$d/source/"
  sed "s/{VERSION}/$VERSION/g" "build/readme/$1.txt" > "$d/READ ME FIRST.txt"
}
for p in linux-x86_64 pi-64bit pi-32bit; do
  pack $p
  cp -a "dist/$p/." "$out/Tinkerie-$VERSION-$p/"
  (cd "$out" && tar czf "Tinkerie-$VERSION-$p.tar.gz" "Tinkerie-$VERSION-$p")
done
for p in mac-apple-silicon mac-intel; do
  pack $p
  cp -a "dist/$p/Tinkerie.app" "$out/Tinkerie-$VERSION-$p/"
  (cd "$out" && zip -qry "Tinkerie-$VERSION-$p.zip" "Tinkerie-$VERSION-$p")
done
pack windows
sed -i 's/$/\r/' "$out/Tinkerie-$VERSION-windows/READ ME FIRST.txt"
cp Tinkerie.exe "$out/Tinkerie-$VERSION-windows/"
(cd "$out" && zip -qr "Tinkerie-$VERSION-windows.zip" "Tinkerie-$VERSION-windows")
cp Tinkerie.exe "$out/"
sed "s/{VERSION}/$VERSION/g" build/release-notes.md > "$out/notes.md"
rm -rf "$src"
for p in linux-x86_64 pi-64bit pi-32bit mac-apple-silicon mac-intel windows; do rm -rf "$out/Tinkerie-$VERSION-$p"; done
ls -l "$out"
