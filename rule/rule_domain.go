package rule

import (
	"context"
	"strings"

	"github.com/sipt/shuttle/dns"
)

const (
	KeyDomainSuffix  = "DOMAIN-SUFFIX"
	KeyDomain        = "DOMAIN"
	KeyDomainKeyword = "DOMAIN-KEYWORD"
)

func init() {
	Register(KeyDomainSuffix, domainSuffixHandle)
	Register(KeyDomain, domainHandle)
	Register(KeyDomainKeyword, domainKeywordHandle)
}
func domainSuffixHandle(_ context.Context, rule *Rule, next Handle, _ dns.Handle) (Handle, error) {
	return func(ctx context.Context, info RequestInfo) *Rule {
		if strings.HasSuffix(info.Domain(), rule.Value) {
			return rule
		}
		return next(ctx, info)
	}, nil
}
func domainHandle(_ context.Context, rule *Rule, next Handle, _ dns.Handle) (Handle, error) {
	return func(ctx context.Context, info RequestInfo) *Rule {
		if len(info.Domain()) == len(rule.Value) && info.Domain() == rule.Value {
			return rule
		}
		return next(ctx, info)
	}, nil
}
func domainKeywordHandle(_ context.Context, rule *Rule, next Handle, _ dns.Handle) (Handle, error) {
	return func(ctx context.Context, info RequestInfo) *Rule {
		if len(info.Domain()) >= len(rule.Value) && strings.Index(info.Domain(), rule.Value) > -1 {
			return rule
		}
		return next(ctx, info)
	}, nil
}
