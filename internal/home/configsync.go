package home

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/configsync"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

func validateSyncedConfig(workDir string) func(context.Context, string) error {
	return func(ctx context.Context, path string) (err error) {
		executable, err := os.Executable()
		if err != nil {
			return fmt.Errorf("finding executable: %w", err)
		}

		cmd := exec.CommandContext(
			ctx,
			executable,
			"--check-config",
			"--config", path,
			"--work-dir", workDir,
			"--no-check-update",
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("checking configuration: %w: %s", err, bytes.TrimSpace(output))
		}

		return nil
	}
}

func scheduleConfigSyncRestart() {
	time.AfterFunc(250*time.Millisecond, func() {
		requestConfigSyncRestart()
	})
}

type configSyncServiceConfig struct {
	Config     *configuration
	Modifier   agh.ConfigModifier
	HTTPClient *http.Client
	Logger     *slog.Logger
	ConfPath   string
	WorkDir    string
	Validate   func(ctx context.Context, path string) (err error)
	Restart    func()
}

type configSyncService struct {
	conf       *configuration
	modifier   agh.ConfigModifier
	httpClient *http.Client
	logger     *slog.Logger
	confPath   string
	workDir    string
	validate   func(ctx context.Context, path string) (err error)
	restart    func()
	syncMu     *sync.Mutex
	statusMu   *sync.Mutex
	statuses   map[string]configSyncPeerResult
	notify     chan struct{}
}

func newConfigSyncService(c *configSyncServiceConfig) (s *configSyncService) {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	s = &configSyncService{
		conf:       c.Config,
		modifier:   c.Modifier,
		httpClient: httpClient,
		logger:     c.Logger,
		confPath:   c.ConfPath,
		workDir:    c.WorkDir,
		validate:   c.Validate,
		restart:    c.Restart,
		syncMu:     &sync.Mutex{},
		statusMu:   &sync.Mutex{},
		statuses:   map[string]configSyncPeerResult{},
		notify:     make(chan struct{}, 1),
	}
	go s.run()

	return s
}

type configSyncSaveRequest struct {
	Role  configsync.Role `json:"role"`
	Links []string        `json:"links"`
}

type configSyncPeerStatus struct {
	Link     string `json:"link"`
	LastSync string `json:"last_sync,omitempty"`
	Error    string `json:"error,omitempty"`
}

type configSyncPeerResult struct {
	when time.Time
	err  string
}

type configSyncStatusResponse struct {
	Role  configsync.Role        `json:"role"`
	Link  string                 `json:"link,omitempty"`
	Peers []configSyncPeerStatus `json:"peers"`
}

func (s *configSyncService) handleStatus(w http.ResponseWriter, r *http.Request) {
	resp := s.status(r.Host, r.TLS != nil)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *configSyncService) status(host string, secure bool) (resp *configSyncStatusResponse) {
	s.conf.RLock()
	conf := s.conf.ConfigSync
	conf.Peers = slices.Clone(conf.Peers)
	s.conf.RUnlock()

	resp = &configSyncStatusResponse{Role: conf.Role, Peers: []configSyncPeerStatus{}}
	switch conf.Role {
	case configsync.RoleSecondary:
		resp.Link = secondaryPairingLink(host, secure, conf.Token)
	case configsync.RolePrimary:
		resp.Peers = s.peerStatuses(conf.Peers)
	}

	return resp
}

func secondaryPairingLink(host string, secure bool, token string) (link string) {
	if token == "" {
		return ""
	}
	scheme := "http"
	if secure {
		scheme = "https"
	}
	link, _ = configsync.BuildPairingLink(scheme+"://"+host, token)

	return link
}

func (s *configSyncService) peerStatuses(peers []configsync.Peer) (statuses []configSyncPeerStatus) {
	for _, peer := range peers {
		link, err := configsync.PairingLink(peer)
		if err != nil {
			continue
		}
		status := configSyncPeerStatus{Link: link}
		s.statusMu.Lock()
		result, ok := s.statuses[peer.Endpoint]
		s.statusMu.Unlock()
		if ok {
			status.LastSync = result.when.UTC().Format(time.RFC3339)
			status.Error = result.err
		}
		statuses = append(statuses, status)
	}

	return statuses
}

func (s *configSyncService) handleSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &configSyncSaveRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		http.Error(w, "decoding request", http.StatusBadRequest)

		return
	}

	next, err := s.configForSave(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	s.conf.Lock()
	s.conf.ConfigSync = next
	s.conf.Unlock()
	if s.modifier != nil {
		s.modifier.Apply(ctx)
	}

	s.handleStatus(w, r)
}

func (s *configSyncService) configForSave(req *configSyncSaveRequest) (next configsync.Config, err error) {
	switch req.Role {
	case configsync.RoleNone:
		return configsync.Config{}, nil
	case configsync.RoleSecondary:
		return s.secondaryConfig()
	case configsync.RolePrimary:
		return primaryConfig(req.Links)
	default:
		return configsync.Config{}, fmt.Errorf("unexpected synchronization role %q", req.Role)
	}
}

func (s *configSyncService) secondaryConfig() (conf configsync.Config, err error) {
	s.conf.RLock()
	token := s.conf.ConfigSync.Token
	s.conf.RUnlock()
	if token == "" {
		token, err = configsync.NewToken(rand.Reader)
		if err != nil {
			return configsync.Config{}, fmt.Errorf("generating pairing token: %w", err)
		}
	}

	return configsync.Config{Role: configsync.RoleSecondary, Token: token}, nil
}

func primaryConfig(links []string) (conf configsync.Config, err error) {
	conf.Role = configsync.RolePrimary
	indices := map[string]int{}
	for _, link := range links {
		peer, parseErr := configsync.ParsePairingLink(link)
		if parseErr != nil {
			return configsync.Config{}, parseErr
		}
		if index, ok := indices[peer.Endpoint]; ok {
			conf.Peers[index] = peer

			continue
		}
		indices[peer.Endpoint] = len(conf.Peers)
		conf.Peers = append(conf.Peers, peer)
	}

	return conf, nil
}

func (s *configSyncService) handleRegenerate(w http.ResponseWriter, r *http.Request) {
	s.conf.Lock()
	if s.conf.ConfigSync.Role != configsync.RoleSecondary {
		s.conf.Unlock()
		http.Error(w, "node is not a secondary server", http.StatusBadRequest)

		return
	}
	token, err := configsync.NewToken(rand.Reader)
	if err == nil {
		s.conf.ConfigSync.Token = token
	}
	s.conf.Unlock()
	if err != nil {
		s.handleError(r.Context(), w, "generating pairing token", err)

		return
	}
	if s.modifier != nil {
		s.modifier.Apply(r.Context())
	}

	s.handleStatus(w, r)
}

func (s *configSyncService) handleSync(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	err := s.syncAll(ctx)
	if err != nil {
		s.handleError(ctx, w, "synchronizing configuration", err)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok"}`)
}

func (s *configSyncService) register(httpReg aghhttp.Registrar, mux *http.ServeMux) {
	httpReg.Register(http.MethodGet, "/control/config_sync/status", s.handleStatus)
	httpReg.Register(http.MethodPost, "/control/config_sync/save", s.handleSave)
	httpReg.Register(http.MethodPost, "/control/config_sync/sync", s.handleSync)
	httpReg.Register(http.MethodPost, "/control/config_sync/regenerate", s.handleRegenerate)
	mux.HandleFunc(configsync.ReceivePath, s.handleReceive)
}

// Notify schedules automatic synchronization after a configuration write.
func (s *configSyncService) Notify() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

func (s *configSyncService) run() {
	for range s.notify {
		time.Sleep(500 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := s.syncAll(ctx)
		cancel()
		if err != nil {
			s.logger.Error("automatic configuration synchronization", slogutil.KeyError, err)
		}
	}
}

func (s *configSyncService) handleReceive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)

		return
	}

	s.conf.RLock()
	role := s.conf.ConfigSync.Role
	token := s.conf.ConfigSync.Token
	s.conf.RUnlock()
	if role != configsync.RoleSecondary || !configsync.ValidToken(token, bearerToken(r)) {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)

		return
	}

	maxSize := int64(configSyncReqBodySzLim.Bytes())
	body, err := io.ReadAll(io.LimitReader(r.Body, maxSize+1))
	if err != nil {
		http.Error(w, "reading configuration", http.StatusBadRequest)

		return
	}
	if int64(len(body)) > maxSize {
		http.Error(w, "configuration is too large", http.StatusRequestEntityTooLarge)

		return
	}

	globalContext.controlLock.Lock()
	defer globalContext.controlLock.Unlock()

	local, err := os.ReadFile(s.confPath)
	if err != nil {
		s.handleError(ctx, w, "reading local configuration", err)

		return
	}
	merged, err := configsync.MergeConfig(body, local)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	err = s.validateAndWrite(ctx, merged)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, `{"status":"ok"}`)
	s.restart()
}

func (s *configSyncService) validateAndWrite(ctx context.Context, data []byte) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(s.confPath), ".config-sync-*.yaml")
	if err != nil {
		return fmt.Errorf("creating temporary configuration: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	err = tmp.Chmod(aghos.DefaultPermFile)
	if err == nil {
		_, err = tmp.Write(data)
	}
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("writing temporary configuration: %w", err)
	}

	err = s.validate(ctx, tmpPath)
	if err != nil {
		return fmt.Errorf("validating configuration: %w", err)
	}

	err = replaceSyncedConfig(tmpPath, s.confPath)
	if err != nil {
		return fmt.Errorf("replacing configuration: %w", err)
	}

	return nil
}

func (s *configSyncService) syncAll(ctx context.Context) (err error) {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()

	s.conf.RLock()
	role := s.conf.ConfigSync.Role
	peers := slices.Clone(s.conf.ConfigSync.Peers)
	s.conf.RUnlock()
	if role != configsync.RolePrimary {
		return nil
	}

	data, err := os.ReadFile(s.confPath)
	if err != nil {
		return fmt.Errorf("reading configuration: %w", err)
	}
	data, err = configsync.ConfigWithoutSync(data)
	if err != nil {
		return err
	}

	errCh := make(chan error, len(peers))
	wg := &sync.WaitGroup{}
	for _, peer := range peers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			peerErr := s.push(ctx, peer, data)
			s.setPeerResult(peer.Endpoint, peerErr)
			if peerErr != nil {
				errCh <- peerErr
			}
		}()
	}
	wg.Wait()
	close(errCh)

	var errs []error
	for peerErr := range errCh {
		errs = append(errs, peerErr)
	}
	return errors.Join(errs...)
}

func (s *configSyncService) setPeerResult(endpoint string, err error) {
	result := configSyncPeerResult{when: time.Now()}
	if err != nil {
		result.err = err.Error()
	}
	s.statusMu.Lock()
	s.statuses[endpoint] = result
	s.statusMu.Unlock()
}

func (s *configSyncService) push(ctx context.Context, peer configsync.Peer, data []byte) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, peer.Endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", peer.Endpoint, err)
	}
	req.Header.Set("Authorization", "Bearer "+peer.Token)
	req.Header.Set("Content-Type", "application/yaml")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("synchronizing %s: %w", peer.Endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("synchronizing %s: status %s", peer.Endpoint, resp.Status)
	}

	return nil
}

func (s *configSyncService) handleError(ctx context.Context, w http.ResponseWriter, msg string, err error) {
	s.logger.ErrorContext(ctx, msg, slogutil.KeyError, err)
	http.Error(w, msg, http.StatusInternalServerError)
}

func bearerToken(r *http.Request) (token string) {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(auth, prefix))
}
