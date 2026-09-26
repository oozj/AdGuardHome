package home

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/configsync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testConfigModifier struct {
	apply func(ctx context.Context)
}

// type check
var _ agh.ConfigModifier = (*testConfigModifier)(nil)

func (m *testConfigModifier) Apply(ctx context.Context) { m.apply(ctx) }

func TestConfigSyncServiceReceive(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	confPath := filepath.Join(dir, "AdGuardHome.yaml")
	local := []byte("schema_version: 31\nusers:\n  - name: secondary\nconfig_sync:\n  role: secondary\n  token: local-secret\n")
	require.NoError(t, os.WriteFile(confPath, local, 0o600))

	restarted := false
	validated := false
	svc := newConfigSyncService(&configSyncServiceConfig{
		Config: &configuration{ConfigSync: configsync.Config{
			Role:  configsync.RoleSecondary,
			Token: "local-secret",
		}},
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConfPath: confPath,
		WorkDir:  dir,
		Validate: func(_ context.Context, path string) (err error) {
			validated = true
			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			assert.Contains(t, string(data), "name: primary")
			assert.Contains(t, string(data), "token: local-secret")

			return nil
		},
		Restart: func() { restarted = true },
	})

	body := []byte("schema_version: 31\nusers:\n  - name: primary\nconfig_sync:\n  role: primary\n")
	req := httptest.NewRequest(http.MethodPost, configsync.ReceivePath, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer local-secret")
	recorder := httptest.NewRecorder()

	svc.handleReceive(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.True(t, validated)
	assert.True(t, restarted)
	got, err := os.ReadFile(confPath)
	require.NoError(t, err)
	assert.Contains(t, string(got), "name: primary")
	assert.Contains(t, string(got), "role: secondary")
	assert.Contains(t, string(got), "token: local-secret")
}

func TestConfigSyncServiceReceiveRejectsBadToken(t *testing.T) {
	t.Parallel()

	svc := newConfigSyncService(&configSyncServiceConfig{
		Config: &configuration{ConfigSync: configsync.Config{
			Role:  configsync.RoleSecondary,
			Token: "local-secret",
		}},
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConfPath: filepath.Join(t.TempDir(), "AdGuardHome.yaml"),
		WorkDir:  t.TempDir(),
		Validate: func(context.Context, string) error { return nil },
		Restart:  func() { t.Fatal("unexpected restart") },
	})

	req := httptest.NewRequest(http.MethodPost, configsync.ReceivePath, bytes.NewReader([]byte("dns: {}\n")))
	req.Header.Set("Authorization", "Bearer wrong")
	recorder := httptest.NewRecorder()

	svc.handleReceive(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestConfigSyncServiceReceiveRejectsMissingConfiguredToken(t *testing.T) {
	t.Parallel()

	svc := newConfigSyncService(&configSyncServiceConfig{
		Config: &configuration{ConfigSync: configsync.Config{
			Role: configsync.RoleSecondary,
		}},
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConfPath: filepath.Join(t.TempDir(), "AdGuardHome.yaml"),
		WorkDir:  t.TempDir(),
		Validate: func(context.Context, string) error { return nil },
		Restart:  func() { t.Fatal("unexpected restart") },
	})

	req := httptest.NewRequest(http.MethodPost, configsync.ReceivePath, bytes.NewReader([]byte("dns: {}\n")))
	recorder := httptest.NewRecorder()
	svc.handleReceive(recorder, req)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestConfigSyncReceiveMiddlewareSettings(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, configsync.ReceivePath, nil)
	assert.True(t, isPublicResource(req.URL.Path))
	assert.Equal(t, configSyncReqBodySzLim, requestBodySizeLimit(req))
}

func TestConfigSyncServicePushesSanitizedConfig(t *testing.T) {
	t.Parallel()

	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer peer-secret", r.Header.Get("Authorization"))
		var err error
		gotBody, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	confPath := filepath.Join(dir, "AdGuardHome.yaml")
	require.NoError(t, os.WriteFile(confPath, []byte(
		"schema_version: 31\ndns:\n  port: 53\nconfig_sync:\n  role: primary\n",
	), 0o600))

	svc := newConfigSyncService(&configSyncServiceConfig{
		Config: &configuration{ConfigSync: configsync.Config{
			Role: configsync.RolePrimary,
			Peers: []configsync.Peer{{
				Endpoint: server.URL,
				Token:    "peer-secret",
			}},
		}},
		HTTPClient: server.Client(),
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConfPath:   confPath,
		WorkDir:    dir,
		Validate:   func(context.Context, string) error { return nil },
		Restart:    func() {},
	})

	err := svc.syncAll(context.Background())
	require.NoError(t, err)
	assert.Contains(t, string(gotBody), "port: 53")
	assert.NotContains(t, string(gotBody), "config_sync")
}

func TestConfigSyncServiceUnavailablePeerDoesNotBlockOthers(t *testing.T) {
	t.Parallel()

	fastCalled := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			select {
			case <-r.Context().Done():
			case <-time.After(time.Second):
			}

			return
		}
		fastCalled <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	confPath := filepath.Join(dir, "AdGuardHome.yaml")
	require.NoError(t, os.WriteFile(confPath, []byte("dns: {}\n"), 0o600))
	svc := newConfigSyncService(&configSyncServiceConfig{
		Config: &configuration{ConfigSync: configsync.Config{
			Role: configsync.RolePrimary,
			Peers: []configsync.Peer{
				{Endpoint: server.URL + "/slow", Token: "slow"},
				{Endpoint: server.URL + "/fast", Token: "fast"},
			},
		}},
		HTTPClient: server.Client(),
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConfPath:   confPath,
		WorkDir:    dir,
		Validate:   func(context.Context, string) error { return nil },
		Restart:    func() {},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := svc.syncAll(ctx)
	assert.Error(t, err)
	select {
	case <-fastCalled:
		// Success.
	default:
		t.Fatal("fast peer was blocked by unavailable peer")
	}
}

func TestConfigSyncServiceRetriesRestartWindow(t *testing.T) {
	t.Parallel()

	requests := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "restarting", http.StatusServiceUnavailable)

			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	confPath := filepath.Join(dir, "AdGuardHome.yaml")
	require.NoError(t, os.WriteFile(confPath, []byte("dns: {}\n"), 0o600))
	svc := newConfigSyncService(&configSyncServiceConfig{
		Config: &configuration{ConfigSync: configsync.Config{
			Role: configsync.RolePrimary,
			Peers: []configsync.Peer{{
				Endpoint: server.URL,
				Token:    "peer-secret",
			}},
		}},
		HTTPClient: server.Client(),
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		ConfPath:   confPath,
		WorkDir:    dir,
		Validate:   func(context.Context, string) error { return nil },
		Restart:    func() {},
		RetryDelay: time.Millisecond,
	})

	err := svc.syncAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int32(2), requests.Load())
}

func TestConfigSyncServiceSaveSecondary(t *testing.T) {
	t.Parallel()

	applied := 0
	conf := &configuration{}
	svc := newConfigSyncService(&configSyncServiceConfig{
		Config:   conf,
		Modifier: &testConfigModifier{apply: func(context.Context) { applied++ }},
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Validate: func(context.Context, string) error { return nil },
		Restart:  func() {},
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/control/config_sync/save",
		strings.NewReader(`{"role":"secondary"}`),
	)
	recorder := httptest.NewRecorder()
	svc.handleSave(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, 1, applied)
	assert.Equal(t, configsync.RoleSecondary, conf.ConfigSync.Role)
	assert.Len(t, conf.ConfigSync.Token, 43)
	assert.Empty(t, conf.ConfigSync.Peers)
}

func TestConfigSyncServiceSavePrimaryParsesLinks(t *testing.T) {
	t.Parallel()

	conf := &configuration{}
	svc := newConfigSyncService(&configSyncServiceConfig{
		Config:   conf,
		Modifier: &testConfigModifier{apply: func(context.Context) {}},
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Validate: func(context.Context, string) error { return nil },
		Restart:  func() {},
	})

	body := `{"role":"primary","links":["http://192.0.2.2:3000` +
		configsync.ReceivePath + `#token=peer-secret"]}`
	req := httptest.NewRequest(http.MethodPost, "/control/config_sync/save", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	svc.handleSave(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, configsync.RolePrimary, conf.ConfigSync.Role)
	require.Len(t, conf.ConfigSync.Peers, 1)
	assert.Equal(t, "http://192.0.2.2:3000"+configsync.ReceivePath, conf.ConfigSync.Peers[0].Endpoint)
	assert.Equal(t, "peer-secret", conf.ConfigSync.Peers[0].Token)
	assert.Empty(t, conf.ConfigSync.Token)
}

func TestPrimaryConfigReplacesTokenForSameEndpoint(t *testing.T) {
	t.Parallel()

	base := "http://192.0.2.2:3000" + configsync.ReceivePath
	conf, err := primaryConfig([]string{base + "#token=old", base + "#token=new"})
	require.NoError(t, err)
	require.Len(t, conf.Peers, 1)
	assert.Equal(t, base, conf.Peers[0].Endpoint)
	assert.Equal(t, "new", conf.Peers[0].Token)
}

func TestConfigSyncServiceStatusBuildsSecondaryLink(t *testing.T) {
	t.Parallel()

	svc := newConfigSyncService(&configSyncServiceConfig{
		Config: &configuration{ConfigSync: configsync.Config{
			Role:  configsync.RoleSecondary,
			Token: "local-secret",
		}},
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Validate: func(context.Context, string) error { return nil },
		Restart:  func() {},
	})

	req := httptest.NewRequest(http.MethodGet, "/control/config_sync/status", nil)
	req.Host = "192.0.2.2:3000"
	recorder := httptest.NewRecorder()
	svc.handleStatus(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "http://192.0.2.2:3000"+configsync.ReceivePath)
	assert.Contains(t, recorder.Body.String(), "token=local-secret")
}
