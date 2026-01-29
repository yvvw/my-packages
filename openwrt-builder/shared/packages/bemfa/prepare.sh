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
		grep "bemfa" |
		grep -m 1 "linux-${my_arch}"
)
wget -q --show-progress -O "bemfa.tar.gz" $download_url && tar -xzf "bemfa.tar.gz" && rm "bemfa.tar.gz"
mv bemfa-*/bemfa . && rm -rf bemfa-* && chmod +x bemfa
