#!/bin/bash

# 流程
# 1. 配置环境变量
# 2. 配置并拉取feeds
# 3. 应用会导致编译失败的补丁 fix
# 4. 生成.config
# 5. 执行自定义脚本 diy
#   5.1 应用自定义补丁，不影响编译 diy
#   5.2 自定义uci-defaults初始化脚本
#   5.3 替换uci-defaults自定义变量
# 8. 编译

set -euo pipefail

set -a
BUILD_TARGET=$BUILD_TARGET
BUILD_VERSION=${BUILD_VERSION:-"25.12"}
BUILDER_DIR=$(dirname "$0")

if [ ! -d "${BUILDER_DIR}/targets/${BUILD_TARGET}" ]; then
	echo "Target '${BUILD_TARGET}' not found"
	exit 1
fi

# feed install
OPTION_FORCE_FEED_INSTALL=${OPTION_FORCE_FEED_INSTALL:-false}
# defconfig only
OPTION_DEFCONFIG_ONLY=${OPTION_DEFCONFIG_ONLY:-false}
# strip modules
OPTION_BUILD_MINIMAL=${OPTION_BUILD_MINIMAL:-false}
# build debug
OPTION_BUILD_DEBUG=${OPTION_BUILD_DEBUG:-false}
# build thread
OPTION_BUILD_THREAD=${OPTION_BUILD_THREAD:-$(nproc)}
# build toolchain only
OPTION_BUILD_TOOLCHAIN_ONLY=${OPTION_BUILD_TOOLCHAIN_ONLY:-false}

GITHUB_ACTIONS=${GITHUB_ACTIONS:-false}

# load settings
SETTINGS_FILE="${BUILDER_DIR}/targets/${BUILD_TARGET}/.settings"
if [ -f "${SETTINGS_FILE}" ]; then
	source "${SETTINGS_FILE}"
fi
unset SETTINGS_FILE

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
set +a

# custom feeds
if ! grep -q "kiddin9" feeds.conf.default; then
	sed -Ei "/telephony|video/d" feeds.conf.default
	echo "src-git kiddin9 https://github.com/kiddin9/kwrt-packages.git;main" >>feeds.conf.default
fi

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

# patch fix
batch_patch "${BUILDER_DIR}/shared/patchs"/fix_*.patch
if [ -d "${BUILDER_DIR}/targets/${BUILD_TARGET}/patchs" ]; then
	batch_patch "${BUILDER_DIR}/targets/${BUILD_TARGET}/patchs"/fix_*.patch
fi

# .config
cat "${BUILDER_DIR}/shared/.config" >.config
cat "${BUILDER_DIR}/targets/${BUILD_TARGET}/.config" >>.config

# diy
if [ -f "${BUILDER_DIR}/shared/diy.sh" ]; then
	. "${BUILDER_DIR}/shared/diy.sh"
fi
if [ -f "${BUILDER_DIR}/targets/${BUILD_TARGET}/diy.sh" ]; then
	. "${BUILDER_DIR}/targets/${BUILD_TARGET}/diy.sh"
fi

# remove patch .rej .orig
find feeds package target -type f \( -name "*.rej" -o -name "*.orig" \) -exec rm -f {} +

# make defconfig
if [ "$OPTION_BUILD_MINIMAL" = "true" ]; then
	make defconfig >/dev/null 2>&1
	sed -i "s,=m,=n,g" .config
fi
make defconfig

if [ "$OPTION_DEFCONFIG_ONLY" = "true" ]; then
	exit 0
fi

if [ "$GITHUB_ACTIONS" = "true" ]; then
	cat .config
fi

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
