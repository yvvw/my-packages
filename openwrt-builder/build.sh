#!/bin/bash
set -euo pipefail

set -a
BUILD_TARGET=$BUILD_TARGET
BUILD_VERSION=${BUILD_VERSION:-"25.12"}
BUILDER_DIR=$(dirname "$0")

if [ ! -d "${BUILDER_DIR}/targets/${BUILD_TARGET}" ]; then
	echo "Target "${BUILD_TARGET}" not found"
	exit 1
fi

# load settings
SETTINGS_FILE="${BUILDER_DIR}/targets/${BUILD_TARGET}/.settings"
[ -f "${SETTINGS_FILE}" ] && . "${SETTINGS_FILE}"
unset SETTINGS_FILE

# build debug
OPTION_BUILD_DEBUG=${OPTION_BUILD_DEBUG:-false}
# build thread
OPTION_BUILD_THREAD=${OPTION_BUILD_THREAD:-$(nproc)}
# strip modules
OPTION_BUILD_MINIMAL=${OPTION_BUILD_MINIMAL:-false}
# build packages
OPTION_BUILD_PACKAGES=${OPTION_BUILD_PACKAGES:-}
# defconfig only
OPTION_DEFCONFIG_ONLY=${OPTION_DEFCONFIG_ONLY:-false}
# build toolchain only
OPTION_BUILD_TOOLCHAIN_ONLY=${OPTION_BUILD_TOOLCHAIN_ONLY:-false}
# feed install
OPTION_FORCE_FEED_INSTALL=${OPTION_FORCE_FEED_INSTALL:-false}

# name
M_NAME="${M_NAME:-openwrt}"
# password
M_PASSWORD="${M_PASSWORD:-password}"
# lan
M_LAN_PREFIX="${M_LAN_PREFIX:-192.168.1}"
# wan pppoe/dhcp
M_WAN_PROTO="${M_WAN_PROTO:-dhcp}"
M_WAN_PPPOE_USER="${M_WAN_PPPOE_USER:-}"
M_WAN_PPPOE_PASS="${M_WAN_PPPOE_PASS:-}"

CONFIG_VERSION_NUMBER=$BUILD_VERSION
CONFIG_VERSION_CODE=$(date +%Y.%m.%d)
CONFIG_VERSION_DIST=$M_NAME
CONFIG_VERSION_MANUFACTURER="${M_NAME} ${BUILD_VERSION}"
CONFIG_VERSION_REPO="https://dl.openwrt.ai/releases/${BUILD_VERSION}"

GITHUB_ACTIONS=${GITHUB_ACTIONS:-false}
set +a

# custom feeds
if ! grep -q "kiddin9" feeds.conf.default; then
	sed -Ei "/telephony|video/d" feeds.conf.default
	echo "src-git kiddin9 https://github.com/kiddin9/kwrt-packages.git;main" >>feeds.conf.default
fi

# custom packages
for pkg in ${OPTION_BUILD_PACKAGES}; do
	[ -d "package/${pkg}" ] && rm -rf "package/${pkg}"
	[ -d "${BUILDER_DIR}/shared/packages/${pkg}" ] &&
		cp -a "${BUILDER_DIR}/shared/packages/${pkg}" "package"
	[ -d "${BUILDER_DIR}/targets/${BUILD_TARGET}/packages/${pkg}" ] &&
		cp -a "${BUILDER_DIR}/targets/${BUILD_TARGET}/packages/${pkg}" "package"
done

# feed install
if [ ! -d "feeds" ] || [ "$OPTION_FORCE_FEED_INSTALL" = "true" ]; then
	./scripts/feeds clean
	./scripts/feeds update -a
	./scripts/feeds install -a
fi

m_patch() {
	local pf="$1"
	[ ! -f "$pf" ] && return 0
	local out=$(patch -p0 --forward --dry-run <"$pf" 2>&1)
	if echo "$out" | grep -q "malformed patch"; then
		echo "Patch failed $pf $out"
		return 1
	elif echo "$out" | grep -q "Reversed"; then
		return 0
	fi
	patch -p0 <"$pf"
}

batch_patch() {
	local patches=("$@")
	for p in "${patches[@]}"; do
		m_patch "$p"
	done
}

# apply fix patch
batch_patch "${BUILDER_DIR}/shared/patchs"/fix_*.patch
if [ -d "${BUILDER_DIR}/targets/${BUILD_TARGET}/patchs" ]; then
	batch_patch "${BUILDER_DIR}/targets/${BUILD_TARGET}/patchs"/fix_*.patch
fi

# merge .config
cat "${BUILDER_DIR}/shared/.config" >.config &&
	cat "${BUILDER_DIR}/targets/${BUILD_TARGET}/.config" >>.config

# diy
[ -f "${BUILDER_DIR}/shared/diy.sh" ] && . "${BUILDER_DIR}/shared/diy.sh"
[ -f "${BUILDER_DIR}/targets/${BUILD_TARGET}/diy.sh" ] && . "${BUILDER_DIR}/targets/${BUILD_TARGET}/diy.sh"

# remove patch .rej .orig
find feeds package target -type f \( -name "*.rej" -o -name "*.orig" \) -exec rm -f {} +

# make defconfig
if [ "$OPTION_BUILD_MINIMAL" = "true" ]; then
	make defconfig >/dev/null 2>&1
	sed -i "s,=m,=n,g" .config
fi
make defconfig

[ "$OPTION_DEFCONFIG_ONLY" = "true" ] && exit 0

[ "$GITHUB_ACTIONS" = "true" ] && cat .config

# toolchain
if [ "$OPTION_BUILD_TOOLCHAIN_ONLY" = "true" ]; then
	if [ "$OPTION_BUILD_DEBUG" = "false" ]; then
		make -j${OPTION_BUILD_THREAD} toolchain/install
	else
		make -j1 V=s toolchain/install
	fi
	exit 0
fi

# build
if [ "$OPTION_BUILD_DEBUG" = "false" ]; then
	make -j${OPTION_BUILD_THREAD}
else
	make -j1 V=s
fi
