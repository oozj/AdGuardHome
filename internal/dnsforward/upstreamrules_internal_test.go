package dnsforward

import (
	"context"
	"io"
	"log/slog"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/upstreamrules"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerSetCustomUpstream_UsesMatchingRuleGroup(t *testing.T) {
	groupConfig := proxy.NewCustomUpstreamConfig(&proxy.UpstreamConfig{}, false, 0, false)
	m, issues := upstreamrules.Compile([]upstreamrules.Group{{
		ID:        7,
		Name:      "foreign",
		Enabled:   true,
		Priority:  1,
		Upstreams: []string{"192.0.2.1"},
		Rules:     "||example.com",
	}})
	require.Empty(t, issues)

	s := &Server{
		conf: ServerConfig{Config: Config{ClientsContainer: EmptyClientsContainer{}}},
		upstreamRules: &upstreamRuleRuntime{
			matcher: m,
			configs: map[uint64]*proxy.CustomUpstreamConfig{7: groupConfig},
		},
	}
	pctx := &proxy.DNSContext{
		Addr: netip.MustParseAddrPort("192.0.2.10:53000"),
		Req:  createTestMessage("www.example.com."),
	}

	_, release := s.setCustomUpstream(
		context.Background(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		pctx,
		"",
	)

	assert.Same(t, groupConfig, pctx.CustomUpstreamConfig)
	assert.False(t, s.serverLock.TryLock())
	release()
	assert.True(t, s.serverLock.TryLock())
	s.serverLock.Unlock()
}

func TestServerSetCustomUpstream_ClientConfigurationTakesPriority(t *testing.T) {
	clientConfig := proxy.NewCustomUpstreamConfig(&proxy.UpstreamConfig{}, false, 0, false)
	groupConfig := proxy.NewCustomUpstreamConfig(&proxy.UpstreamConfig{}, false, 0, false)
	m, issues := upstreamrules.Compile([]upstreamrules.Group{{
		ID:        7,
		Name:      "foreign",
		Enabled:   true,
		Priority:  1,
		Upstreams: []string{"192.0.2.1"},
		Rules:     "*",
	}})
	require.Empty(t, issues)

	s := &Server{
		conf: ServerConfig{Config: Config{ClientsContainer: &clientsContainer{
			OnCustomUpstreamConfig: func(_ string, _ netip.Addr) *proxy.CustomUpstreamConfig {
				return clientConfig
			},
		}}},
		upstreamRules: &upstreamRuleRuntime{
			matcher: m,
			configs: map[uint64]*proxy.CustomUpstreamConfig{7: groupConfig},
		},
	}
	pctx := &proxy.DNSContext{
		Addr: netip.MustParseAddrPort("192.0.2.10:53000"),
		Req:  createTestMessage("www.example.com."),
	}

	_, release := s.setCustomUpstream(
		context.Background(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		pctx,
		"",
	)
	defer release()

	assert.Same(t, clientConfig, pctx.CustomUpstreamConfig)
}
