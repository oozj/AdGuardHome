package configsync

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "go.yaml.in/yaml/v4"
)

func TestPairingLinkRoundTrip(t *testing.T) {
	const (
		baseURL = "http://192.0.2.2:3000"
		token   = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFG"
	)

	link, err := BuildPairingLink(baseURL, token)
	require.NoError(t, err)

	peer, err := ParsePairingLink(link)
	require.NoError(t, err)
	assert.Equal(t, baseURL+ReceivePath, peer.Endpoint)
	assert.Equal(t, token, peer.Token)
}

func TestParsePairingLinkRejectsUnsafeValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		link string
	}{
		{name: "missing_token", link: "http://192.0.2.2:3000" + ReceivePath},
		{name: "wrong_path", link: "http://192.0.2.2:3000/other#token=value"},
		{name: "unsupported_scheme", link: "file:///tmp/config#token=value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParsePairingLink(tc.link)
			assert.Error(t, err)
		})
	}
}

func TestMergePreservesOnlyLocalSyncSettings(t *testing.T) {
	incoming := []byte("schema_version: 31\nusers:\n  - name: primary\ndns:\n  port: 53\nconfig_sync:\n  role: primary\n  peers:\n    - endpoint: http://bad.example\n      token: bad\n")
	local := []byte("schema_version: 31\nusers:\n  - name: secondary\ndns:\n  port: 5353\nconfig_sync:\n  role: secondary\n  token: local-secret\n")

	merged, err := MergeConfig(incoming, local)
	require.NoError(t, err)

	got := map[string]any{}
	require.NoError(t, yaml.Unmarshal(merged, &got))
	assert.Equal(t, []any{map[string]any{"name": "primary"}}, got["users"])
	assert.Equal(t, map[string]any{"port": 53}, got["dns"])
	assert.Equal(t, map[string]any{"role": "secondary", "token": "local-secret"}, got["config_sync"])
}

func TestConfigWithoutSync(t *testing.T) {
	data := []byte("dns:\n  port: 53\nconfig_sync:\n  role: primary\n")

	got, err := ConfigWithoutSync(data)
	require.NoError(t, err)
	assert.NotContains(t, string(got), "config_sync")
	assert.Contains(t, string(got), "port: 53")
}

func TestNewToken(t *testing.T) {
	token, err := NewToken(bytes.NewReader(make([]byte, TokenBytes)))
	require.NoError(t, err)
	assert.Len(t, token, 43)
	assert.NotContains(t, token, "=")
}

func TestValidToken(t *testing.T) {
	assert.True(t, ValidToken("secret", "secret"))
	assert.False(t, ValidToken("secret", "different"))
	assert.False(t, ValidToken("", ""))
}
