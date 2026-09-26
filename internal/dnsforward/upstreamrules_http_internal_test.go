package dnsforward

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/upstreamrules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertUpstreamRuleGroup_AssignsAndKeepsID(t *testing.T) {
	groups, saved, err := upsertUpstreamRuleGroup(nil, upstreamrules.Group{
		Name:      "custom",
		Kind:      groupKindCustom,
		Enabled:   true,
		Priority:  10,
		Upstreams: []string{"192.0.2.1"},
		Rules:     "||example.com",
	})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.NotZero(t, saved.ID)

	saved.Name = "renamed"
	groups, updated, err := upsertUpstreamRuleGroup(groups, saved)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, saved.ID, updated.ID)
	assert.Equal(t, "renamed", groups[0].Name)
}

func TestFetchUpstreamRuleSubscription_DecodesBase64(t *testing.T) {
	want := "[AutoProxy 0.2.9]\n||example.com\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString([]byte(want))))
	}))
	t.Cleanup(srv.Close)

	got, err := fetchUpstreamRuleSubscription(context.Background(), srv.Client(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestValidateUpstreamRuleGroup(t *testing.T) {
	testCases := []struct {
		name  string
		group upstreamrules.Group
		want  string
	}{
		{
			name:  "empty_name",
			group: upstreamrules.Group{Kind: groupKindCustom, Upstreams: []string{"192.0.2.1"}},
			want:  "name",
		},
		{
			name:  "empty_upstreams",
			group: upstreamrules.Group{Name: "group", Kind: groupKindCustom},
			want:  "upstreams",
		},
		{
			name: "bad_subscription_url",
			group: upstreamrules.Group{
				Name:      "group",
				Kind:      groupKindSubscription,
				URL:       "file:///tmp/rules",
				Upstreams: []string{"192.0.2.1"},
			},
			want: "URL",
		},
		{
			name: "no_default_upstream",
			group: upstreamrules.Group{
				Name:      "group",
				Kind:      groupKindCustom,
				Upstreams: []string{"[/example.com/]192.0.2.1"},
			},
			want: "default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUpstreamRuleGroup(tc.group)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestMergeCachedSubscriptionRules(t *testing.T) {
	existing := upstreamrules.Group{
		ID:          1,
		Kind:        groupKindSubscription,
		URL:         "https://example.com/rules.txt",
		Rules:       "||cached.example",
		LastUpdated: "2026-09-21T00:00:00Z",
	}
	edited := existing
	edited.Name = "renamed"
	edited.Rules = ""
	edited.LastUpdated = ""

	merged, ok := mergeCachedSubscriptionRules(edited, existing)
	require.True(t, ok)
	assert.Equal(t, existing.Rules, merged.Rules)
	assert.Equal(t, existing.LastUpdated, merged.LastUpdated)

	edited.URL = "https://example.com/new.txt"
	_, ok = mergeCachedSubscriptionRules(edited, existing)
	assert.False(t, ok)
}

func TestServerHandleUpstreamRuleGroupsGet(t *testing.T) {
	s := &Server{conf: ServerConfig{Config: Config{
		UpstreamDNS: []string{"192.0.2.53"},
		UpstreamRuleGroups: []upstreamrules.Group{
			{
				ID:        1,
				Name:      "custom",
				Kind:      groupKindCustom,
				Enabled:   true,
				Upstreams: []string{"192.0.2.1"},
				Rules:     "||example.com",
			},
			{
				ID:        2,
				Name:      "remote",
				Kind:      groupKindSubscription,
				Enabled:   true,
				Upstreams: []string{"192.0.2.2"},
				Rules:     "||one.example\n||two.example\n",
			},
		},
	}}}
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/control/upstream_rules/status", nil)

	s.handleUpstreamRuleGroupsGet(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	response := &upstreamRuleGroupsStatus{}
	require.NoError(t, json.NewDecoder(recorder.Body).Decode(response))
	assert.Equal(t, []string{"192.0.2.53"}, response.DefaultUpstreams)
	require.Len(t, response.Groups, 2)
	assert.Equal(t, "custom", response.Groups[0].Name)
	assert.Empty(t, response.Groups[1].Rules)
	assert.Equal(t, 2, response.Groups[1].RulesCount)
}

func TestUpstreamRuleGroupForResponse_RedactsSubscriptionRules(t *testing.T) {
	group := upstreamrules.Group{
		ID:    1,
		Name:  "remote",
		Kind:  groupKindSubscription,
		Rules: "||one.example\n||two.example\n",
	}

	response := upstreamRuleGroupForResponse(group)

	assert.Empty(t, response.Rules)
	assert.Equal(t, 2, response.RulesCount)
	assert.NotEmpty(t, group.Rules)
}

func TestDefaultUpstreamAddresses_RemovesDomainSpecificRoutes(t *testing.T) {
	got := defaultUpstreamAddresses([]string{
		"https://one.example/dns-query",
		"[/example.com/]192.0.2.1",
		"  https://two.example/dns-query  ",
	})

	assert.Equal(t, []string{
		"https://one.example/dns-query",
		"  https://two.example/dns-query  ",
	}, got)
}
