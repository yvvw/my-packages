https://github.com/immortalwrt/immortalwrt

https://github.com/kiddin9/Kwrt

```sh
sudo bash -c "bash <(curl -s https://build-scripts.immortalwrt.org/init_build_environment.sh)"

git clone -b openwrt-25.12 --single-branch --filter=blob:none https://github.com/immortalwrt/immortalwrt

# copy builder folder & custom target settings

BUILD_TARGET=target bash builder/build.sh
```
