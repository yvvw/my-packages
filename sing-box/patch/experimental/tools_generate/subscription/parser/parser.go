package parser

import (
	"context"

	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service"
)

var subscriptionParsers = []func(ctx context.Context, content string) ([]option.Outbound, error){
	ParseBoxSubscription,
	ParseClashSubscription,
	ParseSIP008Subscription,
	ParseRawSubscription,
}

func ParseSubscription(ctx context.Context, content string) ([]option.Outbound, error) {
	pctx := service.ContextWithDefaultRegistry(include.Context(ctx))
	var pErr error
	for _, parser := range subscriptionParsers {
		servers, err := parser(pctx, content)
		if len(servers) > 0 {
			return servers, nil
		}
		pErr = E.Errors(pErr, err)
	}
	return nil, E.Cause(pErr, "no servers found")
}
