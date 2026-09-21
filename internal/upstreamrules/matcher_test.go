package upstreamrules

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatcher_DomainAndURLRules(t *testing.T) {
	m, issues := Compile([]Group{{
		ID:        1,
		Name:      "foreign",
		Enabled:   true,
		Priority:  10,
		Upstreams: []string{"https://dns.example/dns-query"},
		Rules: "||example.com\n" +
			"|http://www.dmm.com/netgame\n" +
			"plain.example.org\n",
	}})
	require.Empty(t, issues)

	for _, host := range []string{
		"example.com",
		"www.example.com",
		"www.dmm.com",
		"plain.example.org",
	} {
		match, ok := m.Match(host)
		require.Truef(t, ok, "expected %q to match", host)
		assert.Equal(t, uint64(1), match.GroupID)
	}

	_, ok := m.Match("unrelated.example")
	assert.False(t, ok)
}

func TestMatcher_ExceptionFallsThroughToNextGroup(t *testing.T) {
	m, issues := Compile([]Group{
		{
			ID:        1,
			Name:      "gfw",
			Enabled:   true,
			Priority:  10,
			Upstreams: []string{"https://foreign.example/dns-query"},
			Rules: "||example.com\n@@||safe.example.com\n" +
				"||tokenplus.app\n@@||*.tokenplus.app\n",
		},
		{
			ID:        2,
			Name:      "default",
			Enabled:   true,
			Priority:  20,
			Upstreams: []string{"192.0.2.53"},
			Rules:     "*\n",
		},
	})
	require.Empty(t, issues)

	match, ok := m.Match("blocked.example.com")
	require.True(t, ok)
	assert.Equal(t, uint64(1), match.GroupID)

	match, ok = m.Match("safe.example.com")
	require.True(t, ok)
	assert.Equal(t, uint64(2), match.GroupID)

	match, ok = m.Match("a.b.tokenplus.app")
	require.True(t, ok)
	assert.Equal(t, uint64(2), match.GroupID)
}

func TestMatcher_HostGlobAndURLRegex(t *testing.T) {
	m, issues := Compile([]Group{{
		ID:        1,
		Name:      "patterns",
		Enabled:   true,
		Priority:  1,
		Upstreams: []string{"192.0.2.1"},
		Rules: "||cdn*.i-scmp.com\n" +
			`/^https?:\/\/[^\/]+blogspot\.(.*)/` + "\n",
	}})
	require.Empty(t, issues)

	for _, host := range []string{"cdn12.i-scmp.com", "foo.blogspot.jp"} {
		_, ok := m.Match(host)
		assert.Truef(t, ok, "expected %q to match", host)
	}

	_, ok := m.Match("i-scmp.com")
	assert.False(t, ok)
}

func TestDecodeSubscription_Base64AutoProxyList(t *testing.T) {
	want := "[AutoProxy 0.2.9]\n||example.com\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(want))

	got, err := DecodeSubscription([]byte(encoded))
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCompile_ReportsHostlessRulesWithoutRejectingValidRules(t *testing.T) {
	m, issues := Compile([]Group{{
		ID:        1,
		Name:      "mixed",
		Enabled:   true,
		Priority:  1,
		Upstreams: []string{"192.0.2.1"},
		Rules:     "||valid.example\n[AutoProxy 0.2.9]\n/[/\n",
	}})

	require.Len(t, issues, 1)
	assert.Equal(t, 3, issues[0].Line)
	_, ok := m.Match("valid.example")
	assert.True(t, ok)
}

func TestMatcher_ConvertsHostnameLookaheadException(t *testing.T) {
	m, issues := Compile([]Group{
		{
			ID:        1,
			Name:      "remote",
			Enabled:   true,
			Priority:  1,
			Upstreams: []string{"192.0.2.1"},
			Rules: "||xn--ngstr-lra8j.com\n" +
				`@@/^https?:\/\/(?=.*?(2x3|ni5|j5o))[a-z0-9.-]+\.xn--ngstr-lra8j\.com$` + "\n",
		},
		{
			ID:        2,
			Name:      "default",
			Enabled:   true,
			Priority:  2,
			Upstreams: []string{"192.0.2.2"},
			Rules:     "*",
		},
	})
	require.Empty(t, issues)

	match, ok := m.Match("foo2x3.xn--ngstr-lra8j.com")
	require.True(t, ok)
	assert.Equal(t, uint64(2), match.GroupID)

	match, ok = m.Match("ordinary.xn--ngstr-lra8j.com")
	require.True(t, ok)
	assert.Equal(t, uint64(1), match.GroupID)
}

func TestMatcher_ShortDomainAnchorStopsAtLabelBoundary(t *testing.T) {
	m, issues := Compile([]Group{{
		ID:        1,
		Name:      "short-anchor",
		Enabled:   true,
		Priority:  1,
		Upstreams: []string{"192.0.2.1"},
		Rules:     "||google^",
	}})
	require.Empty(t, issues)

	_, ok := m.Match("google.com")
	assert.True(t, ok)

	_, ok = m.Match("www.google.com")
	assert.True(t, ok)

	_, ok = m.Match("googleevil.com")
	assert.False(t, ok)
}

func TestMatcher_BareShortDomainAnchorMatchesPrefix(t *testing.T) {
	m, issues := Compile([]Group{{
		ID:        1,
		Name:      "bare-short-anchor",
		Enabled:   true,
		Priority:  1,
		Upstreams: []string{"192.0.2.1"},
		Rules:     "||google",
	}})
	require.Empty(t, issues)

	_, ok := m.Match("googleapis.com")
	assert.True(t, ok)

	_, ok = m.Match("www.googleusercontent.com")
	assert.True(t, ok)
}
