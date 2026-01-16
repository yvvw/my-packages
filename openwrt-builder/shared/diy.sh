#!/bin/bash

set -euo pipefail

# patch diy
if [ -d "${BUILDER_DIR}/shared/patchs" ]; then
	batch_patch "${BUILDER_DIR}/shared/patchs"/diy_*.patch
fi
if [ -d "${BUILDER_DIR}/targets/${BUILD_TARGET}/patchs" ]; then
	batch_patch "${BUILDER_DIR}/targets/${BUILD_TARGET}/patchs"/diy_*.patch
fi

# patch tcp bbr
if grep -qE "bbr=y|turboacc=y" .config && [ ! -d "package/turboacc" ]; then
	curl -sSL "https://raw.githubusercontent.com/chenmozhijin/turboacc/luci/add_turboacc.sh" | bash
fi

# custom uci-defaults
UCI_TARGET="package/base-files/files/etc/uci-defaults"
rm -rf "${UCI_TARGET}" && git restore "${UCI_TARGET}"
copy_uci() {
	uci_dir="$1"
	target="$2"
	[ ! -d "${uci_dir}" ] && return 0
	for fp in "${uci_dir}"/*; do
		[ -f "$fp" ] || continue
		fn="${fp##*/}"
		cp "$fp" "${UCI_TARGET}/${fn}-${target}"
	done
}
copy_uci "${BUILDER_DIR}/shared/files/etc/uci-defaults" "shared"
copy_uci "${BUILDER_DIR}/targets/${BUILD_TARGET}/files/etc/uci-defaults" "${BUILD_TARGET}"

for name in ${!M_@}; do
	value="${!name}"
	sed -i "s,<${name}>,${value},g" "${UCI_TARGET}"/*
done
