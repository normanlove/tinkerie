#!/bin/sh
# Adds Tinkerie to your applications menu (Linux and Raspberry Pi OS).
# Run it from the folder that holds the tinkerie program: ./install.sh
# To remove: ./install.sh --remove
set -e
here="$(cd "$(dirname "$0")" && pwd)"
apps="${XDG_DATA_HOME:-$HOME/.local/share}/applications"
file="$apps/tinkerie.desktop"
if [ "$1" = "--remove" ]; then
  rm -f "$file"
  echo "Removed Tinkerie from the applications menu."
  exit 0
fi
chmod +x "$here/tinkerie"
mkdir -p "$apps"
cat > "$file" <<EOF
[Desktop Entry]
Type=Application
Name=Tinkerie
Comment=Free maker tools by nrmluvdev: Wirie, Makie, Tunie, Flashie, Scopie, Meterie, Tracie, Codie
Exec="$here/tinkerie"
Icon=$here/tinkerie.png
Terminal=false
Categories=Development;Electronics;Engineering;
EOF
chmod +x "$file"
echo "Added Tinkerie to the applications menu (look under Programming or Development)."
