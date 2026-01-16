include $(TOPDIR)/rules.mk

PKG_NAME:=sing-box
PKG_VERSION:=1.13.0

include $(INCLUDE_DIR)/package.mk

define Package/$(PKG_NAME)
  SECTION:=net
  CATEGORY:=Network
  TITLE:=sing-box
  DEPENDS:=+ca-bundle +kmod-tun +kmod-nft-queue
endef

define Package/$(PKG_NAME)/conffiles
/etc/sing-box/
/etc/config/sing-box
endef

define Package/$(PKG_NAME)/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/sing-box $(1)/usr/bin/sing-box
	$(INSTALL_DIR) $(1)/etc/init.d/
	$(INSTALL_BIN) ./files/sing-box.init $(1)/etc/init.d/sing-box
	$(INSTALL_DIR) $(1)/etc/config/
	$(INSTALL_CONF) ./files/sing-box.conf $(1)/etc/config/sing-box
	$(INSTALL_DIR) $(1)/etc/sing-box
endef

define Build/Prepare
	$(call Build/Prepare/Default)
	$(INSTALL_DIR) $(PKG_BUILD_DIR)
	$(CP) ./prepare.sh $(PKG_BUILD_DIR)/
	chmod +x $(PKG_BUILD_DIR)/prepare.sh
	$(PKG_BUILD_DIR)/prepare.sh $(ARCH) $(PKG_BUILD_DIR)
endef

define Build/Compile
endef

$(eval $(call BuildPackage,$(PKG_NAME)))
