package dnsforward

import (
	"context"
	"fmt"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghslog"
	"github.com/AdguardTeam/AdGuardHome/internal/upstreamrules"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// upstreamRuleRuntime is an immutable set of compiled rules and prepared
// per-group upstream configurations.
type upstreamRuleRuntime struct {
	matcher *upstreamrules.Matcher
	configs map[uint64]*proxy.CustomUpstreamConfig
}

func (r *upstreamRuleRuntime) close() {
	if r == nil {
		return
	}

	for _, conf := range r.configs {
		_ = conf.Close()
	}
}

func (r *upstreamRuleRuntime) customConfig(qname string) (conf *proxy.CustomUpstreamConfig) {
	if r == nil || r.matcher == nil {
		return nil
	}

	match, ok := r.matcher.Match(qname)
	if !ok {
		return nil
	}

	return r.configs[match.GroupID]
}

func (s *Server) prepareUpstreamRuleSettings(ctx context.Context, boot upstream.Resolver) {
	if s.upstreamRules != nil {
		s.upstreamRules.close()
	}

	matcher, issues := upstreamrules.Compile(s.conf.UpstreamRuleGroups)
	for _, issue := range issues {
		s.logger.WarnContext(
			ctx,
			"compiling upstream rule",
			"line", issue.Line,
			slogutil.KeyError, issue.Message,
		)
	}

	runtime := &upstreamRuleRuntime{
		matcher: matcher,
		configs: map[uint64]*proxy.CustomUpstreamConfig{},
	}
	for _, group := range s.conf.UpstreamRuleGroups {
		if !group.Enabled || len(group.Upstreams) == 0 {
			continue
		}

		uc, err := proxy.ParseUpstreamsConfig(group.Upstreams, &upstream.Options{
			Logger:       aghslog.NewForUpstream(s.baseLogger, aghslog.UpstreamTypeMain),
			Bootstrap:    boot,
			Timeout:      s.conf.UpstreamTimeout,
			HTTPVersions: aghnet.UpstreamHTTPVersions(s.conf.UseHTTP3Upstreams),
			PreferIPv6:   s.conf.BootstrapPreferIPv6,
			RootCAs:      s.conf.TLSv12Roots,
			CipherSuites: s.conf.TLSCiphers,
		})
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"preparing upstream rule group",
				"group", group.Name,
				slogutil.KeyError, fmt.Errorf("parsing upstreams: %w", err),
			)

			continue
		}

		runtime.configs[group.ID] = proxy.NewCustomUpstreamConfig(
			uc,
			s.conf.CacheEnabled,
			int(s.conf.CacheSize),
			s.conf.EDNSClientSubnet.Enabled,
		)
	}

	s.upstreamRules = runtime
}
