#!/bin/sh
set -eu

repo="elliotnorth/Scimitar"
version="${SCIMITAR_VERSION:-latest}"

fail() {
	echo "scimitar: $*" >&2
	exit 1
}

[ "$(uname -s)" = Linux ] || fail "Scimitar runs on Linux only."
command -v systemctl >/dev/null 2>&1 || fail "Scimitar needs a system with systemd."
case "$(uname -m)" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) fail "unsupported CPU: $(uname -m)" ;;
esac

if [ "$version" = latest ]; then
	base="https://github.com/$repo/releases/latest/download"
else
	base="https://github.com/$repo/releases/download/$version"
fi

if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -q "$1" -O "$2"; }
else
	fail "install curl or wget first."
fi

if [ "$(id -u)" -eq 0 ]; then
	sudo=""
elif command -v sudo >/dev/null 2>&1; then
	sudo="sudo"
else
	fail "run this as root or install sudo."
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading Scimitar for $arch..."
fetch "$base/scimitar-linux-$arch" "$tmp/scimitar-linux-$arch"
fetch "$base/checksums.txt" "$tmp/checksums.txt"
(cd "$tmp" && grep " scimitar-linux-$arch\$" checksums.txt | sha256sum -c --quiet -) || fail "checksum mismatch, nothing was installed."
chmod +x "$tmp/scimitar-linux-$arch"

if [ -x /usr/local/bin/scimitar ] && [ -f /etc/scimitar/scimitar.json ]; then
	echo "Scimitar is already installed. Updating it."
	$sudo "$tmp/scimitar-linux-$arch" update
	exit 0
fi

$sudo "$tmp/scimitar-linux-$arch" install </dev/tty
