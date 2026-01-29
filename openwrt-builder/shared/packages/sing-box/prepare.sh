#!/bin/sh
set -e

arch="$1"
output="$2"

case "$arch" in
x86_64)
	my_arch="amd64"
	;;
aarch64)
	my_arch="arm64"
	;;
*)
	echo "Unsupported $arch"
	exit 1
	;;
esac

cd "$output"

download_url=$(
	wget -q -O- "https://api.github.com/repos/yvvw/my-packages/releases" |
		jq -r ".[].assets[].browser_download_url" |
		grep "sing-box" |
		grep -m 1 "linux-${my_arch}"
)
wget -q --show-progress -O "sing-box.tar.gz" $download_url && tar -xzf "sing-box.tar.gz" && rm "sing-box.tar.gz"
mv sing-box-*/sing-box . && rm -rf sing-box-* && chmod +x sing-box
