include $(TOPDIR)/rules.mk

PKG_NAME:=bemfa
PKG_VERSION:=1.0.0

include $(INCLUDE_DIR)/package.mk

define Package/$(PKG_NAME)
  SECTION:=utils
  CATEGORY:=Utilities
  TITLE:=Bemfa
endef

define Package/$(PKG_NAME)/conffiles
/etc/config/bemfa
endef

define Package/$(PKG_NAME)/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/bemfa $(1)/usr/bin/bemfa
	$(INSTALL_DIR) $(1)/etc/init.d/
	$(INSTALL_BIN) ./files/bemfa.init $(1)/etc/init.d/bemfa
	$(INSTALL_DIR) $(1)/etc/config/
	$(INSTALL_CONF) ./files/bemfa.conf $(1)/etc/config/bemfa
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
