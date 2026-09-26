// Package configsync contains configuration synchronization primitives.
package configsync

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"

	yaml "go.yaml.in/yaml/v4"
)

// ReceivePath is the unauthenticated-by-session configuration receiver path.
const ReceivePath = "/control/config_sync/receive"

// TokenBytes is the number of random bytes in a pairing token.
const TokenBytes = 32

// Role is a configuration synchronization role.
type Role string

const (
	// RoleNone disables configuration synchronization.
	RoleNone Role = ""

	// RolePrimary pushes configuration to secondary servers.
	RolePrimary Role = "primary"

	// RoleSecondary receives configuration from a primary server.
	RoleSecondary Role = "secondary"
)

// Peer is a parsed secondary-server pairing link.
type Peer struct {
	Endpoint string `json:"endpoint" yaml:"endpoint"`
	Token    string `json:"-" yaml:"token"`
}

// Config is the node-local configuration synchronization section.
type Config struct {
	Role  Role   `json:"role" yaml:"role,omitempty"`
	Token string `json:"-" yaml:"token,omitempty"`
	Peers []Peer `json:"-" yaml:"peers,omitempty"`
}

// BuildPairingLink builds a copyable secondary-server pairing link.
func BuildPairingLink(baseURL, token string) (link string, err error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("parsing base URL: %w", err)
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return "", fmt.Errorf("base URL must use HTTP or HTTPS without user information")
	}
	if token == "" {
		return "", fmt.Errorf("token must not be empty")
	}

	u.Path = ReceivePath
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = url.Values{"token": []string{token}}.Encode()

	return u.String(), nil
}

// ParsePairingLink parses a secondary-server pairing link.
func ParsePairingLink(link string) (peer Peer, err error) {
	u, err := url.Parse(strings.TrimSpace(link))
	if err != nil {
		return Peer{}, fmt.Errorf("parsing pairing link: %w", err)
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return Peer{}, fmt.Errorf("pairing link must use HTTP or HTTPS without user information")
	}
	if u.Path != ReceivePath {
		return Peer{}, fmt.Errorf("pairing link has unexpected path %q", u.Path)
	}

	fragment, err := url.ParseQuery(u.Fragment)
	if err != nil {
		return Peer{}, fmt.Errorf("parsing pairing token: %w", err)
	}
	token := fragment.Get("token")
	if token == "" {
		return Peer{}, fmt.Errorf("pairing link has no token")
	}

	u.Fragment = ""
	u.RawFragment = ""

	return Peer{Endpoint: u.String(), Token: token}, nil
}

// PairingLink returns the copyable pairing link for peer.
func PairingLink(peer Peer) (link string, err error) {
	u, err := url.Parse(peer.Endpoint)
	if err != nil {
		return "", fmt.Errorf("parsing peer endpoint: %w", err)
	}
	if u.Path != ReceivePath || peer.Token == "" {
		return "", fmt.Errorf("peer is incomplete")
	}

	u.Fragment = url.Values{"token": []string{peer.Token}}.Encode()

	return u.String(), nil
}

// ConfigWithoutSync returns data without the node-local config_sync section.
func ConfigWithoutSync(data []byte) (clean []byte, err error) {
	m, err := yamlMap(data)
	if err != nil {
		return nil, err
	}

	delete(m, "config_sync")
	clean, err = yaml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshalling configuration: %w", err)
	}

	return clean, nil
}

// MergeConfig returns the incoming configuration with local synchronization
// settings preserved.
func MergeConfig(incoming, local []byte) (merged []byte, err error) {
	incomingMap, err := yamlMap(incoming)
	if err != nil {
		return nil, fmt.Errorf("parsing incoming configuration: %w", err)
	}
	localMap, err := yamlMap(local)
	if err != nil {
		return nil, fmt.Errorf("parsing local configuration: %w", err)
	}

	delete(incomingMap, "config_sync")
	if syncConfig, ok := localMap["config_sync"]; ok {
		incomingMap["config_sync"] = syncConfig
	}

	merged, err = yaml.Marshal(incomingMap)
	if err != nil {
		return nil, fmt.Errorf("marshalling merged configuration: %w", err)
	}

	return merged, nil
}

func yamlMap(data []byte) (m map[string]any, err error) {
	m = map[string]any{}
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		return nil, fmt.Errorf("parsing configuration YAML: %w", err)
	}

	return m, nil
}

// NewToken returns a new URL-safe pairing token using random.
func NewToken(random io.Reader) (token string, err error) {
	b := make([]byte, TokenBytes)
	_, err = io.ReadFull(random, b)
	if err != nil {
		return "", fmt.Errorf("reading token randomness: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ValidToken reports whether actual equals expected in constant time.
func ValidToken(expected, actual string) (ok bool) {
	if expected == "" || actual == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}
