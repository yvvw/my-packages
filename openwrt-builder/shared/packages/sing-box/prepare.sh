#!/bin/sh
set -e

ARCH="$1"
DEST="$2"

case "$ARCH" in
x86_64)
	MY_ARCH="amd64"
	;;
aarch64)
	MY_ARCH="arm64"
	;;
*)
	echo "Unsupported $ARCH"
	exit 1
	;;
esac

cd "$DEST"

name="sing-box"
download_url=$(
	wget -q -O- "https://api.github.com/repos/yvvw/my-packages/releases" |
		jq -r ".[].assets[].browser_download_url" |
		grep "${name}" |
		grep -m 1 "linux-${MY_ARCH}"
)
wget -O "${name}.tar.gz" $download_url && tar -xzf "${name}.tar.gz" && rm "${name}.tar.gz"
mv ${name}-*/${name} . && chmod +x ${name} && rm -rf ${name}-*
