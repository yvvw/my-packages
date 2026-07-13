package parser

import (
	"context"
	"strconv"
	"strings"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/format"
	"github.com/sagernet/sing/common/json/badoption"
	N "github.com/sagernet/sing/common/network"
	"gopkg.in/yaml.v3"
)

type clashConfig struct {
	Proxies []map[string]any `yaml:"proxies"`
}

func ParseClashSubscription(_ context.Context, content string) ([]option.Outbound, error) {
	var config clashConfig
	err := yaml.Unmarshal([]byte(content), &config)
	if err != nil {
		return nil, E.Cause(err, "parse clash config")
	}
	var outbounds []option.Outbound
	for _, proxyMapping := range config.Proxies {
		var outbound option.Outbound
		outbound.Tag = clashString(proxyMapping, "name")
		switch clashString(proxyMapping, "type") {
		case "anytls":
			tlsOptions := &option.OutboundTLSOptions{
				Enabled:    true,
				ServerName: clashString(proxyMapping, "sni"),
				Insecure:   true,
				ALPN:       clashStringList(proxyMapping["alpn"]),
			}
			if clientFingerprint := clashString(proxyMapping, "client-fingerprint"); clientFingerprint != "" {
				tlsOptions.UTLS = &option.OutboundUTLSOptions{
					Enabled:     true,
					Fingerprint: clientFingerprint,
				}
			}
			echOptions := clashMap(proxyMapping["ech-opts"])
			if clashBool(echOptions, "enable") {
				tlsOptions.ECH = &option.OutboundECHOptions{
					Enabled:         true,
					QueryServerName: clashString(echOptions, "query-server-name"),
				}
				if echConfig := clashString(echOptions, "config"); echConfig != "" {
					tlsOptions.ECH.Config = badoption.Listable[string]{echConfig}
				}
			}
			if certificate, privateKey := clashString(proxyMapping, "certificate"), clashString(proxyMapping, "private-key"); certificate != "" && privateKey != "" {
				tlsOptions.ClientCertificate = badoption.Listable[string]{certificate}
				tlsOptions.ClientKey = badoption.Listable[string]{privateKey}
			}
			outbound.Type = C.TypeAnyTLS
			outbound.Options = &option.AnyTLSOutboundOptions{
				DialerOptions: option.DialerOptions{},
				ServerOptions: option.ServerOptions{
					Server:     clashString(proxyMapping, "server"),
					ServerPort: uint16(clashInt(proxyMapping, "port")),
				},
				OutboundTLSOptionsContainer: option.OutboundTLSOptionsContainer{
					TLS: tlsOptions,
				},
				Password: clashString(proxyMapping, "password"),
				IdleSessionCheckInterval: badoption.Duration(
					time.Duration(clashInt(proxyMapping, "idle-session-check-interval")) * time.Second,
				),
				IdleSessionTimeout: badoption.Duration(
					time.Duration(clashInt(proxyMapping, "idle-session-timeout")) * time.Second,
				),
				MinIdleSession: clashInt(proxyMapping, "min-idle-session"),
			}
		case "ss", "shadowsocks":
			pluginName := clashPluginName(clashString(proxyMapping, "plugin"))
			if pluginName == "shadow-tls" {
				continue
			}
			outbound.Type = C.TypeShadowsocks
			outbound.Options = &option.ShadowsocksOutboundOptions{
				ServerOptions: option.ServerOptions{
					Server:     clashString(proxyMapping, "server"),
					ServerPort: uint16(clashInt(proxyMapping, "port")),
				},
				Password:      clashString(proxyMapping, "password"),
				Method:        clashShadowsocksCipher(clashString(proxyMapping, "cipher")),
				Plugin:        pluginName,
				PluginOptions: clashPluginOptions(clashString(proxyMapping, "plugin"), clashMap(proxyMapping["plugin-opts"])),
				Network:       clashNetworks(clashBool(proxyMapping, "udp")),
			}
		case "ssr", "shadowsocksr":
			outbound.Type = C.TypeShadowsocksR
			outbound.Options = &option.ShadowsocksROutboundOptions{
				ServerOptions: option.ServerOptions{
					Server:     clashString(proxyMapping, "server"),
					ServerPort: uint16(clashInt(proxyMapping, "port")),
				},
				Password:      clashString(proxyMapping, "password"),
				Method:        clashShadowsocksCipher(clashString(proxyMapping, "cipher")),
				Protocol:      clashString(proxyMapping, "protocol"),
				ProtocolParam: clashString(proxyMapping, "protocol-param"),
				Obfs:          clashString(proxyMapping, "obfs"),
				ObfsParam:     clashString(proxyMapping, "obfs-param"),
				Network:       clashNetworks(clashBool(proxyMapping, "udp")),
			}
		case "trojan":
			outbound.Type = C.TypeTrojan
			outbound.Options = &option.TrojanOutboundOptions{
				ServerOptions: option.ServerOptions{
					Server:     clashString(proxyMapping, "server"),
					ServerPort: uint16(clashInt(proxyMapping, "port")),
				},
				Password: clashString(proxyMapping, "password"),
				OutboundTLSOptionsContainer: option.OutboundTLSOptionsContainer{
					TLS: &option.OutboundTLSOptions{
						Enabled:    true,
						ALPN:       clashStringList(proxyMapping["alpn"]),
						ServerName: clashString(proxyMapping, "sni"),
						Insecure:   clashBool(proxyMapping, "skip-cert-verify"),
					},
				},
				Transport: clashTransport(clashString(proxyMapping, "network"), nil, nil, clashMap(proxyMapping["grpc-opts"]), clashMap(proxyMapping["ws-opts"])),
				Network:   clashNetworks(clashBool(proxyMapping, "udp")),
			}
		case "vmess":
			outbound.Type = C.TypeVMess
			outbound.Options = &option.VMessOutboundOptions{
				ServerOptions: option.ServerOptions{
					Server:     clashString(proxyMapping, "server"),
					ServerPort: uint16(clashInt(proxyMapping, "port")),
				},
				UUID:     clashString(proxyMapping, "uuid"),
				Security: clashString(proxyMapping, "cipher"),
				AlterId:  clashInt(proxyMapping, "alterId", "alter-id"),
				OutboundTLSOptionsContainer: option.OutboundTLSOptionsContainer{
					TLS: &option.OutboundTLSOptions{
						Enabled:    clashBool(proxyMapping, "tls"),
						ServerName: clashString(proxyMapping, "servername", "server-name", "sni"),
						Insecure:   clashBool(proxyMapping, "skip-cert-verify"),
					},
				},
				Transport: clashTransport(clashString(proxyMapping, "network"), clashMap(proxyMapping["http-opts"]), clashMap(proxyMapping["h2-opts"]), clashMap(proxyMapping["grpc-opts"]), clashMap(proxyMapping["ws-opts"])),
				Network:   clashNetworks(clashBool(proxyMapping, "udp")),
			}
		case "socks5":
			if clashBool(proxyMapping, "tls") {
				continue
			}
			outbound.Type = C.TypeSOCKS
			outbound.Options = &option.SOCKSOutboundOptions{
				ServerOptions: option.ServerOptions{
					Server:     clashString(proxyMapping, "server"),
					ServerPort: uint16(clashInt(proxyMapping, "port")),
				},
				Username: clashString(proxyMapping, "username", "user"),
				Password: clashString(proxyMapping, "password"),
				Network:  clashNetworks(clashBool(proxyMapping, "udp")),
			}
		case "http":
			if clashBool(proxyMapping, "tls") {
				continue
			}
			outbound.Type = C.TypeHTTP
			outbound.Options = &option.HTTPOutboundOptions{
				ServerOptions: option.ServerOptions{
					Server:     clashString(proxyMapping, "server"),
					ServerPort: uint16(clashInt(proxyMapping, "port")),
				},
				Username: clashString(proxyMapping, "username", "user"),
				Password: clashString(proxyMapping, "password"),
			}
		default:
			continue
		}
		outbounds = append(outbounds, outbound)
	}
	if len(outbounds) > 0 {
		return outbounds, nil
	}
	return nil, E.New("no servers found")
}

func clashShadowsocksCipher(cipher string) string {
	switch cipher {
	case "dummy":
		return "none"
	}
	return cipher
}

func clashNetworks(udpEnabled bool) option.NetworkList {
	if !udpEnabled {
		return N.NetworkTCP
	}
	return ""
}

func clashPluginName(plugin string) string {
	switch plugin {
	case "obfs":
		return "obfs-local"
	}
	return plugin
}

type shadowsocksPluginOptionsBuilder map[string]any

func (o shadowsocksPluginOptionsBuilder) Build() string {
	var opts []string
	for key, value := range o {
		if value == nil {
			continue
		}
		opts = append(opts, format.ToString(key, "=", value))
	}
	return strings.Join(opts, ";")
}

func clashPluginOptions(plugin string, opts map[string]any) string {
	options := make(shadowsocksPluginOptionsBuilder)
	switch plugin {
	case "obfs":
		options["obfs"] = opts["mode"]
		options["obfs-host"] = opts["host"]
	case "v2ray-plugin":
		options["mode"] = opts["mode"]
		options["tls"] = opts["tls"]
		options["host"] = opts["host"]
		options["path"] = opts["path"]
	}
	return options.Build()
}

func clashTransport(network string, httpOpts map[string]any, h2Opts map[string]any, grpcOpts map[string]any, wsOpts map[string]any) *option.V2RayTransportOptions {
	switch network {
	case "http":
		return &option.V2RayTransportOptions{
			Type: C.V2RayTransportTypeHTTP,
			HTTPOptions: option.V2RayHTTPOptions{
				Method:  clashString(httpOpts, "method"),
				Path:    clashFirstString(httpOpts["path"]),
				Headers: clashHTTPHeaders(httpOpts["headers"]),
			},
		}
	case "h2":
		return &option.V2RayTransportOptions{
			Type: C.V2RayTransportTypeHTTP,
			HTTPOptions: option.V2RayHTTPOptions{
				Path: clashString(h2Opts, "path"),
				Host: clashStringList(h2Opts["host"]),
			},
		}
	case "grpc":
		return &option.V2RayTransportOptions{
			Type: C.V2RayTransportTypeGRPC,
			GRPCOptions: option.V2RayGRPCOptions{
				ServiceName: clashString(grpcOpts, "grpc-service-name", "service-name"),
			},
		}
	case "ws":
		return &option.V2RayTransportOptions{
			Type: C.V2RayTransportTypeWebsocket,
			WebsocketOptions: option.V2RayWebsocketOptions{
				Path:                clashString(wsOpts, "path"),
				Headers:             clashHTTPHeaders(wsOpts["headers"]),
				MaxEarlyData:        uint32(clashInt(wsOpts, "max-early-data")),
				EarlyDataHeaderName: clashString(wsOpts, "early-data-header-name"),
			},
		}
	default:
		return nil
	}
}

func clashMap(value any) map[string]any {
	switch typedValue := value.(type) {
	case map[string]any:
		return typedValue
	case map[any]any:
		result := make(map[string]any, len(typedValue))
		for key, value := range typedValue {
			result[format.ToString(key)] = value
		}
		return result
	default:
		return nil
	}
}

func clashString(mapping map[string]any, keys ...string) string {
	for _, key := range keys {
		switch value := mapping[key].(type) {
		case string:
			return value
		case int:
			return strconv.Itoa(value)
		case int64:
			return strconv.FormatInt(value, 10)
		case uint64:
			return strconv.FormatUint(value, 10)
		case float64:
			return strconv.FormatInt(int64(value), 10)
		case bool:
			return strconv.FormatBool(value)
		}
	}
	return ""
}

func clashBool(mapping map[string]any, keys ...string) bool {
	for _, key := range keys {
		switch value := mapping[key].(type) {
		case bool:
			return value
		case string:
			result, _ := strconv.ParseBool(value)
			return result
		case int:
			return value != 0
		case int64:
			return value != 0
		case uint64:
			return value != 0
		case float64:
			return value != 0
		}
	}
	return false
}

func clashInt(mapping map[string]any, keys ...string) int {
	for _, key := range keys {
		switch value := mapping[key].(type) {
		case int:
			return value
		case int64:
			return int(value)
		case uint64:
			return int(value)
		case float64:
			return int(value)
		case string:
			result, _ := strconv.Atoi(value)
			return result
		}
	}
	return 0
}

func clashStringList(value any) badoption.Listable[string] {
	switch typedValue := value.(type) {
	case []string:
		return typedValue
	case []any:
		result := make([]string, 0, len(typedValue))
		for _, value := range typedValue {
			if stringValue := clashAnyString(value); stringValue != "" {
				result = append(result, stringValue)
			}
		}
		return result
	case string:
		if typedValue != "" {
			return badoption.Listable[string]{typedValue}
		}
	}
	return nil
}

func clashFirstString(value any) string {
	list := clashStringList(value)
	if len(list) > 0 {
		return list[0]
	}
	return clashAnyString(value)
}

func clashHTTPHeaders(value any) badoption.HTTPHeader {
	headerMap := clashMap(value)
	if len(headerMap) == 0 {
		return nil
	}
	headers := make(badoption.HTTPHeader, len(headerMap))
	for key, value := range headerMap {
		headers[key] = clashStringList(value)
	}
	return headers
}

func clashAnyString(value any) string {
	switch typedValue := value.(type) {
	case string:
		return typedValue
	case int:
		return strconv.Itoa(typedValue)
	case int64:
		return strconv.FormatInt(typedValue, 10)
	case uint64:
		return strconv.FormatUint(typedValue, 10)
	case float64:
		return strconv.FormatInt(int64(typedValue), 10)
	case bool:
		return strconv.FormatBool(typedValue)
	default:
		return ""
	}
}
