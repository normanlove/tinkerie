#!/bin/sh
# Tinkerie web — installer for Raspberry Pi (any model, 32- or 64-bit,
# Raspberry Pi OS / Raspbian Buster and newer, and other Debian or Ubuntu
# systems). Puts the Tinkerie web apps on this Pi's web server.
#
#   sh install-pi.sh                  install or update (asks before changing anything big)
#   sh install-pi.sh --yes            same, without questions (uses the defaults)
#   sh install-pi.sh --webroot DIR    install into DIR/tinkerie (default: your web server's site folder)
#   sh install-pi.sh --path NAME      use /NAME/ instead of /tinkerie/
#   sh install-pi.sh --domain NAME    also get a free HTTPS certificate for NAME (Let's Encrypt)
#   sh install-pi.sh --version VER    install a particular release (default: the newest)
#   sh install-pi.sh --uninstall      remove Tinkerie web again
#
# The apps are plain files: nothing runs on the Pi except the web server you
# already have (Apache or nginx; Apache is installed if there's neither).
# They need https:// to start, except on the Pi itself (http://localhost).
# Made by nrmluvdev — https://tinkerie.net
set -eu

REPO="normanlove/tinkerie"
NAME="tinkerie"
WEBROOT=""
DOMAIN=""
VERSION=""
YES=0
UNINSTALL=0

say()  { printf '%s\n' "$*"; }
step() { printf '\n\033[1m%s\033[0m\n' "$*"; }
warn() { printf '\033[33m%s\033[0m\n' "$*" >&2; }
die()  { printf '\033[31mStopped: %s\033[0m\n' "$*" >&2; exit 1; }

while [ $# -gt 0 ]; do
  case "$1" in
    --yes|-y) YES=1 ;;
    --webroot) WEBROOT="${2:?--webroot needs a folder}"; shift ;;
    --path) NAME="${2:?--path needs a name}"; shift ;;
    --domain) DOMAIN="${2:?--domain needs a name}"; shift ;;
    --version) VERSION="${2:?--version needs a version}"; shift ;;
    --uninstall) UNINSTALL=1 ;;
    -h|--help) sed -n '2,20p' "$0"; exit 0 ;;
    *) die "unknown option $1 (try --help)" ;;
  esac
  shift
done
case "$NAME" in *[!a-zA-Z0-9._-]*|"") die "--path may only use letters, numbers, dots, dashes and underscores" ;; esac

ask() { # ask "question" -> 0 for yes
  [ "$YES" = 1 ] && return 0
  [ -t 0 ] || [ -r /dev/tty ] || return 1
  printf '%s [Y/n] ' "$1"
  read -r a </dev/tty || return 1
  case "$a" in n*|N*) return 1 ;; *) return 0 ;; esac
}

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
  command -v sudo >/dev/null 2>&1 || die "run this as root, or install sudo"
  SUDO="sudo"
fi
here=$(cd "$(dirname "$0")" && pwd)

# ---------------------------------------------------------------- web server
SERVER=""
if command -v apache2ctl >/dev/null 2>&1 || [ -d /etc/apache2 ]; then SERVER=apache
elif command -v nginx >/dev/null 2>&1 || [ -d /etc/nginx ]; then SERVER=nginx
elif command -v lighttpd >/dev/null 2>&1; then SERVER=lighttpd
fi

docroot_apache() { # the DocumentRoot of the enabled site that serves HTTPS, else the first one
  for f in /etc/apache2/sites-enabled/*; do
    [ -e "$f" ] || continue
    if grep -qiE '<VirtualHost[^>]*:443' "$f" 2>/dev/null; then
      d=$(grep -iE '^[[:space:]]*DocumentRoot' "$f" | head -n1 | awk '{print $2}' | tr -d '"')
      [ -n "$d" ] && { echo "$d"; return; }
    fi
  done
  for f in /etc/apache2/sites-enabled/*; do
    [ -e "$f" ] || continue
    d=$(grep -iE '^[[:space:]]*DocumentRoot' "$f" 2>/dev/null | head -n1 | awk '{print $2}' | tr -d '"')
    [ -n "$d" ] && { echo "$d"; return; }
  done
  echo /var/www/html
}
docroot_nginx() {
  d=$(grep -rhE '^[[:space:]]*root[[:space:]]' /etc/nginx/sites-enabled/ /etc/nginx/conf.d/ 2>/dev/null | head -n1 | awk '{print $2}' | tr -d ';"')
  echo "${d:-/var/www/html}"
}

if [ -z "$WEBROOT" ]; then
  case "$SERVER" in
    apache) WEBROOT=$(docroot_apache) ;;
    nginx) WEBROOT=$(docroot_nginx) ;;
    lighttpd) WEBROOT=$(grep -E '^server.document-root' /etc/lighttpd/lighttpd.conf 2>/dev/null | sed 's/.*= *"\(.*\)"/\1/'); WEBROOT=${WEBROOT:-/var/www/html} ;;
    *) WEBROOT=/var/www/html ;;
  esac
fi
DEST="$WEBROOT/$NAME"

# ----------------------------------------------------------------- uninstall
if [ "$UNINSTALL" = 1 ]; then
  step "Removing Tinkerie web"
  [ -f "$DEST/version.json" ] || [ ! -e "$DEST" ] || die "$DEST doesn't look like Tinkerie web; not removing it"
  $SUDO rm -rf "$DEST"
  if [ -f /etc/apache2/conf-available/tinkerie-web.conf ]; then
    $SUDO a2disconf -q tinkerie-web >/dev/null 2>&1 || true
    $SUDO rm -f /etc/apache2/conf-available/tinkerie-web.conf
    $SUDO systemctl reload apache2 2>/dev/null || $SUDO service apache2 reload || true
  fi
  if [ -f /etc/nginx/conf.d/tinkerie-web.conf ]; then
    $SUDO rm -f /etc/nginx/conf.d/tinkerie-web.conf
    $SUDO systemctl reload nginx 2>/dev/null || true
  fi
  say "Done. Each browser keeps its own saved work until you clear that site's data."
  exit 0
fi

step "Tinkerie web installer"
model=$(uname -m)
[ -r /proc/device-tree/model ] && model=$(tr -d '\0' </proc/device-tree/model)
say "This Pi: $model, $(. /etc/os-release 2>/dev/null && echo "$PRETTY_NAME" || uname -sr)"
say "Web server: ${SERVER:-none yet}. Apps will go in $DEST (https://…/$NAME/)."

if [ -z "$SERVER" ]; then
  command -v apt-get >/dev/null 2>&1 || die "no web server found, and this system has no apt-get to install one. Install Apache or nginx, then run this again."
  ask "No web server is installed. Install Apache?" || die "a web server is needed"
  step "Installing Apache"
  $SUDO apt-get update -qq
  $SUDO apt-get install -y -qq apache2
  SERVER=apache
  [ -d "$WEBROOT" ] || WEBROOT=/var/www/html
  DEST="$WEBROOT/$NAME"
fi
[ -d "$WEBROOT" ] || die "the web folder $WEBROOT doesn't exist (use --webroot)"

# ------------------------------------------------------------------- files
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
if [ -z "$VERSION" ] && [ -f "$here/tinkerie/version.json" ]; then
  step "Using the files next to this installer"
  cp -R "$here/tinkerie" "$tmp/tinkerie"
else
  fetch() { if command -v curl >/dev/null 2>&1; then curl -fsSL "$1"; else wget -qO- "$1"; fi; }
  command -v curl >/dev/null 2>&1 || command -v wget >/dev/null 2>&1 || $SUDO apt-get install -y -qq curl
  if [ -z "$VERSION" ]; then
    step "Finding the newest release"
    VERSION=$(fetch "https://api.github.com/repos/$REPO/releases?per_page=10" | grep -o '"tag_name": *"v[^"]*"' | head -n1 | sed 's/.*"v\([^"]*\)"/\1/')
    [ -n "$VERSION" ] || die "couldn't reach GitHub to find the newest release (check the Pi's internet connection)"
  fi
  VERSION=${VERSION#v}
  step "Downloading Tinkerie web $VERSION"
  fetch "https://github.com/$REPO/releases/download/v$VERSION/Tinkerie-web-$VERSION.tar.gz" >"$tmp/web.tgz" \
    || die "couldn't download Tinkerie-web-$VERSION.tar.gz (does that release have a web version?)"
  tar -xzf "$tmp/web.tgz" -C "$tmp"
fi
[ -f "$tmp/tinkerie/version.json" ] && [ -f "$tmp/tinkerie/sw.js" ] || die "the download doesn't contain Tinkerie web"
got=$(sed 's/.*"version": *"\([^"]*\)".*/\1/' "$tmp/tinkerie/version.json")

if [ -e "$DEST" ] && [ ! -f "$DEST/version.json" ]; then
  die "$DEST already exists and isn't Tinkerie web — choose another name with --path"
fi
step "Installing Tinkerie web $got"
owner=$(stat -c '%U:%G' "$WEBROOT" 2>/dev/null || echo root:root)
$SUDO rm -rf "$DEST.new"
$SUDO cp -R "$tmp/tinkerie" "$DEST.new"
$SUDO chown -R "$owner" "$DEST.new" 2>/dev/null || true
$SUDO find "$DEST.new" -type d -exec chmod 755 {} + && $SUDO find "$DEST.new" -type f -exec chmod 644 {} +
$SUDO rm -rf "$DEST.old"
[ -e "$DEST" ] && $SUDO mv "$DEST" "$DEST.old"
$SUDO mv "$DEST.new" "$DEST"
$SUDO rm -rf "$DEST.old"

# ------------------------------------------------- server settings (wasm etc.)
step "Setting up $SERVER for the apps"
case "$SERVER" in
apache)
  $SUDO tee /etc/apache2/conf-available/tinkerie-web.conf >/dev/null <<EOF
# Tinkerie web (nrmluvdev): WebAssembly type, compression, and fresh app files.
AddType application/wasm .wasm
<IfModule mod_deflate.c>
  AddOutputFilterByType DEFLATE application/wasm application/javascript text/javascript text/css text/html application/json image/svg+xml
</IfModule>
<Directory "$DEST">
  Options -Indexes
  <IfModule mod_headers.c>
    <FilesMatch "\.(html|js|css|json|wasm)$">
      Header set Cache-Control "no-cache"
    </FilesMatch>
    Header set X-Content-Type-Options "nosniff"
  </IfModule>
</Directory>
EOF
  $SUDO a2enmod -q headers deflate mime >/dev/null 2>&1 || true
  $SUDO a2enconf -q tinkerie-web >/dev/null
  $SUDO apache2ctl configtest >/dev/null 2>&1 || { $SUDO a2disconf -q tinkerie-web; die "Apache didn't accept the settings (apache2ctl configtest); nothing else was changed"; }
  $SUDO systemctl reload apache2 2>/dev/null || $SUDO service apache2 reload
  ;;
nginx)
  if ! grep -rqs 'application/wasm' /etc/nginx/mime.types; then
    $SUDO tee /etc/nginx/conf.d/tinkerie-web.conf >/dev/null <<'EOF'
# Tinkerie web (nrmluvdev): WebAssembly type and compression.
types { application/wasm wasm; }
gzip_types application/wasm application/javascript text/css application/json image/svg+xml;
EOF
  else
    $SUDO tee /etc/nginx/conf.d/tinkerie-web.conf >/dev/null <<'EOF'
# Tinkerie web (nrmluvdev): compression for the app files.
gzip_types application/wasm application/javascript text/css application/json image/svg+xml;
EOF
  fi
  $SUDO nginx -t >/dev/null 2>&1 || { $SUDO rm -f /etc/nginx/conf.d/tinkerie-web.conf; die "nginx didn't accept the settings; nothing else was changed"; }
  $SUDO systemctl reload nginx 2>/dev/null || $SUDO service nginx reload
  ;;
lighttpd)
  grep -qs 'wasm' /etc/lighttpd/lighttpd.conf /etc/lighttpd/conf-enabled/* 2>/dev/null || \
    warn "Add  \".wasm\" => \"application/wasm\"  to lighttpd's mimetype.assign, then restart it."
  ;;
esac

# --------------------------------------------------------------------- https
has_https=0
if [ "$SERVER" = apache ] && grep -qsiE '<VirtualHost[^>]*:443' /etc/apache2/sites-enabled/*; then has_https=1; fi
if [ "$SERVER" = nginx ] && grep -qsE 'listen[^;]*443' /etc/nginx/sites-enabled/* /etc/nginx/conf.d/*; then has_https=1; fi

if [ -n "$DOMAIN" ]; then
  step "Getting an HTTPS certificate for $DOMAIN"
  say "Port 80 must reach this Pi from the internet, and $DOMAIN must point at it."
  $SUDO apt-get install -y -qq certbot "python3-certbot-$SERVER" >/dev/null
  $SUDO certbot "--$SERVER" -d "$DOMAIN" --non-interactive --agree-tos --register-unsafely-without-email --redirect \
    && has_https=1 || warn "certbot couldn't get a certificate; see the messages above."
elif [ "$has_https" = 0 ] && [ "$SERVER" = apache ] && [ "$WEBROOT" = /var/www/html ]; then
  say ""
  say "The apps only start on secure (https://) pages. This server has no HTTPS yet."
  if ask "Turn on HTTPS with a self-signed certificate for your home network? (Browsers will warn once; choose Advanced > Continue)"; then
    $SUDO apt-get install -y -qq ssl-cert >/dev/null 2>&1 || true
    [ -f /etc/ssl/certs/ssl-cert-snakeoil.pem ] || $SUDO make-ssl-cert generate-default-snakeoil --force-overwrite
    $SUDO a2enmod -q ssl >/dev/null
    $SUDO a2ensite -q default-ssl >/dev/null
    $SUDO apache2ctl configtest >/dev/null 2>&1 && { $SUDO systemctl reload apache2 2>/dev/null || $SUDO service apache2 reload; has_https=1; } \
      || { $SUDO a2dissite -q default-ssl; warn "Apache didn't accept HTTPS; left it off."; }
  fi
fi

# ------------------------------------------------------------------- done
host=$(hostname 2>/dev/null || echo raspberrypi)
ips=$(hostname -I 2>/dev/null | tr ' ' '\n' | grep -E '^[0-9]+\.' | head -n3 | tr '\n' ' ')
step "Tinkerie web $got is installed"
if [ -n "$DOMAIN" ]; then say "  https://$DOMAIN/$NAME/"; fi
if [ "$has_https" = 1 ]; then
  say "  https://$host.local/$NAME/"
  for ip in $ips; do say "  https://$ip/$NAME/"; done
else
  say "  On this Pi:  http://localhost/$NAME/"
  say "  From other computers it needs https://. Run again with --domain yourname.example to get a"
  say "  free certificate, or answer yes to the self-signed question."
fi
say ""
say "Best in Chrome or Edge. To update later, run this installer again."
