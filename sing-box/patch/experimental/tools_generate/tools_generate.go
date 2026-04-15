package tools_generate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/sagernet/sing-box/option"

	S "github.com/sagernet/sing-box/experimental/tools_generate/subscription"
	U "github.com/sagernet/sing-box/experimental/tools_generate/utils"
	"github.com/sagernet/sing/common/json"
)

type Config struct {
	SubscriptionList []subscriptionConfig `toml:"subscriptions"`
	SingBox          singBoxConfig        `toml:"sing-box"`
}

func Parse(configBytes []byte) (*Config, error) {
	config := &Config{}
	_, err := toml.Decode(string(configBytes), config)
	if err != nil {
		return nil, err
	}
	for idx := range config.SingBox.RuleSetList {
		if config.SingBox.RuleSetList[idx].Type == "" {
			config.SingBox.RuleSetList[idx].Type = "remote"
		}
		if config.SingBox.RuleSetList[idx].Format == "" {
			config.SingBox.RuleSetList[idx].Format = "binary"
		}
	}
	for idx := range config.SingBox.RouteRuleList {
		if config.SingBox.RouteRuleList[idx].Action == "" {
			config.SingBox.RouteRuleList[idx].Action = "route"
		}
	}
	return config, nil
}

type subscriptionConfig struct {
	Name            string   `toml:"name" json:"name"`
	URL             string   `toml:"url" json:"url"`
	Content         string   `toml:"content" json:"content"`
	DefaultOutbound string   `toml:"default" json:"default"`
	Keywords        []string `toml:"keywords" json:"keywords,omitempty"`
}

type singBoxConfig struct {
	Template         string   `toml:"template" json:"template"`
	Output           string   `toml:"output" json:"output"`
	Gateway          string   `toml:"gateway" json:"gateway"`
	ClashPort        int      `toml:"clash_port" json:"clash_port"`
	DefaultOutbound  string   `toml:"default" json:"default"`
	AutoOutboundList []string `toml:"auto_outbounds" json:"auto_outbounds,omitempty"`
	IncludeServer    bool     `toml:"include_server" json:"include_server,omitempty"`

	RuleSetList   []singBoxRuleSetConfig       `toml:"rule_set" json:"rule_set,omitempty"`
	DirectRule    singBoxRouteRuleDirectConfig `toml:"direct_rule" json:"direct_rule"`
	ProxyRule     singBoxRouteRuleProxyConfig  `toml:"proxy_rule" json:"proxy_rule"`
	BlockRule     singBoxRouteRuleBlockConfig  `toml:"block_rule" json:"block_rule"`
	DNSRuleList   []singBoxDNSRuleConfig       `toml:"dns_rules" json:"dns_rules,omitempty"`
	RouteRuleList []singBoxRouteRuleConfig     `toml:"route_rules" json:"route_rules,omitempty"`
}

type singBoxRuleSetConfig struct {
	Tag            string `toml:"tag" json:"tag"`
	Url            string `toml:"url" json:"url"`
	Type           string `toml:"-" json:"type"`
	Format         string `toml:"-" json:"format"`
	HTTPClient     string `toml:"http_client" json:"http_client,omitempty"`
	DownloadDetour string `toml:"download_detour" json:"download_detour,omitempty"`
}

type singBoxDNSRuleConfig struct {
	Server       string   `toml:"server" json:"server"`
	Domain       []string `toml:"domain" json:"domain,omitempty"`
	DomainSuffix []string `toml:"domain_suffix" json:"domain_suffix,omitempty"`
	RuleSet      []string `toml:"rule_set" json:"rule_set,omitempty"`
}

type singBoxRouteRuleBaseConfig struct {
	Domain       []string `toml:"domain" json:"domain,omitempty"`
	DomainSuffix []string `toml:"domain_suffix" json:"domain_suffix,omitempty"`
	RuleSet      []string `toml:"rule_set" json:"rule_set,omitempty"`
}

type singBoxRouteRuleDirectConfig struct {
	singBoxRouteRuleBaseConfig
	IPCIDR []string `toml:"ip_cidr" json:"ip_cidr,omitempty"`
}

type singBoxRouteRuleProxyConfig struct {
	singBoxRouteRuleDirectConfig
}

type singBoxRouteRuleBlockConfig struct {
	singBoxRouteRuleBaseConfig
}

type singBoxRouteRuleConfig struct {
	singBoxRouteRuleProxyConfig
	Action   string `toml:"action" json:"action"`
	Outbound string `toml:"outbound" json:"outbound"`
}

func readTemplateHeuristic(configName string, name string) ([]byte, error) {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		base := filepath.Base(name)
		name = path.Join(filepath.Dir(configName), base)
		if _, err := os.Stat(name); os.IsNotExist(err) {
			name = base
		}
	}
	return os.ReadFile(name)
}

func GenerateSingBoxConfig(configName string, config *Config) ([]byte, error) {
	tmplBytes, err := readTemplateHeuristic(configName, config.SingBox.Template)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("sing-box").
		Funcs(template.FuncMap{
			"MarshalArray":  U.MarshalArrayF,
			"ConcatStrings": U.Concat[string],
		}).
		Parse(string(tmplBytes))
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	subscriptionList, err := getSubscriptions(ctx, config)
	if err != nil {
		return nil, err
	}

	var outbounds []string
	var outboundTags []string
	var outboundDomains []string
	var outboundGroups []map[string]any
	var outboundGroupTags []string

	for idx, subCfg := range config.SubscriptionList {
		var subOutboundTags []string
		subscriptions := subscriptionList[idx]
		for _, subscription := range subscriptions {
			if len(subCfg.Keywords) > 0 && !slices.ContainsFunc(subCfg.Keywords, func(kw string) bool {
				return strings.Contains(subscription.Tag, kw)
			}) {
				continue
			}

			subscription.Tag = subCfg.Name + "-" + subscription.Tag
			outbound, err := subscription.MarshalJSONContext(ctx)
			if err != nil {
				return nil, err
			}
			outbounds = append(outbounds, string(outbound))
			subOutboundTags = append(subOutboundTags, subscription.Tag)
			if config.SingBox.IncludeServer {
				var options struct {
					Server string `json:"server"`
				}
				err := json.Unmarshal(outbound, &options)
				if err != nil {
					continue
				}
				outboundDomains = append(outboundDomains, options.Server)
			}
		}
		sort.Strings(subOutboundTags)
		outboundTags = append(outboundTags, subOutboundTags...)

		subscriptionOutboundGroupTag := "out-" + subCfg.Name
		outboundGroupTags = append(outboundGroupTags, subscriptionOutboundGroupTag)

		defaultSubscriptionOutboundTag := subCfg.Name + "-" + subCfg.DefaultOutbound
		if !slices.Contains(subOutboundTags, defaultSubscriptionOutboundTag) {
			defaultSubscriptionOutboundTag = subOutboundTags[0]
		}

		outboundGroups = append(outboundGroups, map[string]any{
			"Tag":                subscriptionOutboundGroupTag,
			"DefaultOutboundTag": defaultSubscriptionOutboundTag,
			"OutboundTags":       subOutboundTags,
		})
	}

	var autoOutbounds []string
	{
		for _, tag := range config.SingBox.AutoOutboundList {
			if U.Contains(outboundTags, tag) {
				autoOutbounds = append(autoOutbounds, tag)
			}
		}
		autoOutbounds = U.Unique(autoOutbounds)
		if len(autoOutbounds) > 1 {
			outboundGroupTags = append([]string{"out-proxy-auto"}, outboundGroupTags...)
		}
	}

	tmplBuffer := &bytes.Buffer{}
	err = tmpl.Execute(tmplBuffer, struct {
		DNSRules   []string
		RouteRules []string
		RuleSet    []string

		Gateway   string
		ClashPort int

		OutboundTags       []string
		DefaultOutboundTag string
		OutboundGroups     []map[string]any
		Outbounds          []string
		AutoOutbounds      []string

		DirectDomains []string
		ProxyDomains  []string
		BlockDomains  []string

		DirectDomainSuffixes []string
		ProxyDomainSuffixes  []string
		BlockDomainSuffixes  []string

		DirectIPs []string
		ProxyIPs  []string

		DirectRuleSet []string
		ProxyRuleSet  []string
		BlockRuleSet  []string
	}{
		DNSRules:   marshal(config.SingBox.DNSRuleList),
		RouteRules: marshal(config.SingBox.RouteRuleList),
		RuleSet:    marshal(config.SingBox.RuleSetList),

		Gateway:   config.SingBox.Gateway,
		ClashPort: config.SingBox.ClashPort,

		OutboundTags: outboundGroupTags,
		DefaultOutboundTag: func() string {
			if !slices.Contains(outboundGroupTags, config.SingBox.DefaultOutbound) {
				return outboundGroupTags[0]
			}
			return config.SingBox.DefaultOutbound
		}(),
		OutboundGroups: outboundGroups,
		Outbounds:      outbounds,
		AutoOutbounds:  autoOutbounds,

		DirectDomains: func() []string {
			var subscriptionDomains []string
			for _, subscription := range config.SubscriptionList {
				if subscription.URL == "" {
					continue
				}
				u, err := url.Parse(subscription.URL)
				if err == nil {
					subscriptionDomains = append(subscriptionDomains, u.Hostname())
				}
			}
			return U.Unique(config.SingBox.DirectRule.Domain, subscriptionDomains, outboundDomains)
		}(),
		ProxyDomains: config.SingBox.ProxyRule.Domain,
		BlockDomains: config.SingBox.BlockRule.Domain,

		DirectDomainSuffixes: config.SingBox.DirectRule.DomainSuffix,
		ProxyDomainSuffixes:  config.SingBox.ProxyRule.DomainSuffix,
		BlockDomainSuffixes:  config.SingBox.BlockRule.DomainSuffix,

		DirectIPs: config.SingBox.DirectRule.IPCIDR,
		ProxyIPs:  config.SingBox.ProxyRule.IPCIDR,

		DirectRuleSet: config.SingBox.DirectRule.RuleSet,
		ProxyRuleSet:  config.SingBox.ProxyRule.RuleSet,
		BlockRuleSet:  config.SingBox.BlockRule.RuleSet,
	})
	if err != nil {
		return nil, err
	}

	return tmplBuffer.Bytes(), nil
}

func getSubscriptions(ctx context.Context, config *Config) (outboundList [][]option.Outbound, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	outboundList = make([][]option.Outbound, len(config.SubscriptionList))
	wg := sync.WaitGroup{}
	wg.Add(len(config.SubscriptionList))
	for idx, subConfig := range config.SubscriptionList {
		go func() {
			defer wg.Done()

			var outbounds []option.Outbound
			var sErr error
			if subConfig.URL != "" {
				outbounds, sErr = S.Get(ctx, subConfig.URL)
			} else if subConfig.Content != "" {
				outbounds, sErr = S.Get(ctx, subConfig.Content)
			} else {
				sErr = errors.New("empty url and content")
			}
			if sErr == nil && len(outbounds) == 0 {
				sErr = fmt.Errorf("empty outbounds %v", subConfig)
			}
			if sErr == nil {
				outboundList[idx] = outbounds
				return
			}
			if sErr != nil && !errors.Is(sErr, context.Canceled) {
				err = sErr
				cancel()
			}
		}()
	}
	wg.Wait()
	return
}

func marshal(list any) (results []string) {
	value := reflect.ValueOf(list)
	length := value.Len()
	for i := 0; i < length; i++ {
		bs, err := json.Marshal(value.Index(i).Interface())
		if err != nil {
			continue
		}
		results = append(results, string(bs))
	}
	return
}
