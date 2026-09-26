package dnsforward

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/upstreamrules"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
)

type upstreamRuleGroupsStatus struct {
	DefaultUpstreams []string              `json:"default_upstreams"`
	Groups           []upstreamrules.Group `json:"groups"`
	Issues           []upstreamrules.Issue `json:"issues"`
}

type upstreamRuleGroupIDRequest struct {
	ID uint64 `json:"id"`
}

type upstreamRuleTestRequest struct {
	Domain string `json:"domain"`
}

type upstreamRuleTestResponse struct {
	Matched   bool     `json:"matched"`
	GroupID   uint64   `json:"group_id,omitempty"`
	GroupName string   `json:"group_name,omitempty"`
	Upstreams []string `json:"upstreams,omitempty"`
}

const (
	groupKindCustom       = "custom"
	groupKindSubscription = "subscription"
	maxSubscriptionSize   = 16 * 1024 * 1024
)

func (s *Server) handleUpstreamRuleGroupsGet(w http.ResponseWriter, r *http.Request) {
	groups := s.currentUpstreamRuleGroups()
	if groups == nil {
		groups = []upstreamrules.Group{}
	}
	_, issues := upstreamrules.Compile(groups)
	for i := range groups {
		groups[i] = upstreamRuleGroupForResponse(groups[i])
	}
	s.serverLock.RLock()
	defaultUpstreams, err := s.conf.loadUpstreams(r.Context(), s.logger)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "loading default upstreams", "error", err)
		defaultUpstreams = slices.Clone(s.conf.UpstreamDNS)
	}
	s.serverLock.RUnlock()
	defaultUpstreams = defaultUpstreamAddresses(defaultUpstreams)

	aghhttp.WriteJSONResponseOK(r.Context(), s.logger, w, r, &upstreamRuleGroupsStatus{
		DefaultUpstreams: defaultUpstreams,
		Groups:           groups,
		Issues:           issues,
	})
}

func upstreamRuleGroupForResponse(group upstreamrules.Group) (response upstreamrules.Group) {
	response = group
	response.RulesCount = countUpstreamRuleLines(group.Rules)
	if response.Kind == groupKindSubscription {
		response.Rules = ""
	}

	return response
}

func defaultUpstreamAddresses(upstreams []string) (addresses []string) {
	addresses = slices.Clone(upstreams)

	return slices.DeleteFunc(addresses, func(address string) bool {
		return strings.HasPrefix(strings.TrimSpace(address), "[")
	})
}

func countUpstreamRuleLines(rules string) (count int) {
	for _, line := range strings.Split(rules, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "#") ||
			(strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")) {
			continue
		}
		count++
	}

	return count
}

func upsertUpstreamRuleGroup(
	groups []upstreamrules.Group,
	group upstreamrules.Group,
) (updated []upstreamrules.Group, saved upstreamrules.Group, err error) {
	err = validateUpstreamRuleGroup(group)
	if err != nil {
		return nil, upstreamrules.Group{}, err
	}

	updated = slices.Clone(groups)
	if group.ID == 0 {
		for _, existing := range groups {
			group.ID = max(group.ID, existing.ID)
		}
		group.ID++
		updated = append(updated, group)

		return updated, group, nil
	}

	for i := range updated {
		if updated[i].ID == group.ID {
			updated[i] = group

			return updated, group, nil
		}
	}

	return nil, upstreamrules.Group{}, fmt.Errorf("upstream rule group %d does not exist", group.ID)
}

func fetchUpstreamRuleSubscription(
	ctx context.Context,
	client *http.Client,
	rulesURL string,
) (rules string, err error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rulesURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating subscription request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading subscription: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("downloading subscription: status %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSubscriptionSize+1))
	if err != nil {
		return "", fmt.Errorf("reading subscription: %w", err)
	}
	if len(body) > maxSubscriptionSize {
		return "", fmt.Errorf("subscription exceeds %d bytes", maxSubscriptionSize)
	}

	rules, err = upstreamrules.DecodeSubscription(body)
	if err != nil {
		return "", fmt.Errorf("decoding subscription: %w", err)
	}

	return rules, nil
}

func validateUpstreamRuleGroup(group upstreamrules.Group) (err error) {
	if strings.TrimSpace(group.Name) == "" {
		return fmt.Errorf("name must not be empty")
	}
	if err = validateUpstreamRuleGroupResolvers(group.Upstreams); err != nil {
		return err
	}

	return validateUpstreamRuleGroupKind(group)
}

func validateUpstreamRuleGroupResolvers(upstreams []string) (err error) {
	if len(upstreams) == 0 {
		return fmt.Errorf("upstreams must not be empty")
	}
	for i, upstreamAddr := range upstreams {
		if strings.TrimSpace(upstreamAddr) == "" {
			return fmt.Errorf("upstreams: address at index %d is empty", i)
		}
	}
	uc, err := proxy.ParseUpstreamsConfig(upstreams, &upstream.Options{})
	if err != nil {
		return fmt.Errorf("upstreams: %w", err)
	}
	defer func() { _ = uc.Close() }()
	if len(uc.Upstreams) == 0 {
		return fmt.Errorf("upstreams must include at least one default resolver")
	}

	return nil
}

func validateUpstreamRuleGroupKind(group upstreamrules.Group) (err error) {
	switch group.Kind {
	case groupKindCustom:
		return nil
	case groupKindSubscription:
		u, parseErr := url.Parse(group.URL)
		if parseErr != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("subscription URL must use HTTP or HTTPS")
		}

		return nil
	default:
		return fmt.Errorf("unexpected group kind %q", group.Kind)
	}
}

func mergeCachedSubscriptionRules(
	group upstreamrules.Group,
	existing upstreamrules.Group,
) (merged upstreamrules.Group, ok bool) {
	if group.ID == 0 || existing.ID != group.ID ||
		group.Kind != groupKindSubscription || existing.Kind != groupKindSubscription ||
		group.URL != existing.URL {
		return upstreamrules.Group{}, false
	}

	group.Rules = existing.Rules
	group.LastUpdated = existing.LastUpdated
	group.LastError = existing.LastError

	return group, true
}

func cloneUpstreamRuleGroups(groups []upstreamrules.Group) (cloned []upstreamrules.Group) {
	cloned = slices.Clone(groups)
	for i := range cloned {
		cloned[i].Upstreams = slices.Clone(cloned[i].Upstreams)
	}

	return cloned
}

func (s *Server) currentUpstreamRuleGroups() (groups []upstreamrules.Group) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return cloneUpstreamRuleGroups(s.conf.UpstreamRuleGroups)
}

func (s *Server) replaceUpstreamRuleGroups(
	ctx context.Context,
	groups []upstreamrules.Group,
) (err error) {
	old := s.currentUpstreamRuleGroups()
	s.serverLock.Lock()
	s.conf.UpstreamRuleGroups = cloneUpstreamRuleGroups(groups)
	s.serverLock.Unlock()
	s.conf.ConfModifier.Apply(ctx)

	err = s.Reconfigure(ctx, nil)
	if err == nil {
		return nil
	}

	s.serverLock.Lock()
	s.conf.UpstreamRuleGroups = old
	s.serverLock.Unlock()
	s.conf.ConfModifier.Apply(ctx)
	if rollbackErr := s.Reconfigure(ctx, nil); rollbackErr != nil {
		s.logger.ErrorContext(ctx, "rolling back upstream rule groups", "error", rollbackErr)
	}

	return err
}

func (s *Server) prepareUpstreamRuleGroup(
	ctx context.Context,
	group upstreamrules.Group,
	currentGroups []upstreamrules.Group,
) (prepared upstreamrules.Group, err error) {
	if group.Kind != groupKindSubscription {
		group.URL = ""

		return group, nil
	}

	index := slices.IndexFunc(currentGroups, func(existing upstreamrules.Group) bool {
		return existing.ID == group.ID
	})
	if index >= 0 {
		if merged, ok := mergeCachedSubscriptionRules(group, currentGroups[index]); ok {
			return merged, nil
		}
	}

	group.Rules, err = fetchUpstreamRuleSubscription(ctx, s.conf.HTTPClient, group.URL)
	if err != nil {
		return upstreamrules.Group{}, err
	}
	group.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	group.LastError = ""

	return group, nil
}

func (s *Server) handleUpstreamRuleGroupSave(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s.upstreamRuleGroupsMu.Lock()
	defer s.upstreamRuleGroupsMu.Unlock()

	group := upstreamrules.Group{}
	err := json.NewDecoder(r.Body).Decode(&group)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusBadRequest, "decoding request: %s", err)

		return
	}

	currentGroups := s.currentUpstreamRuleGroups()
	group, err = s.prepareUpstreamRuleGroup(ctx, group, currentGroups)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusBadGateway, "%s", err)

		return
	}

	groups, saved, err := upsertUpstreamRuleGroup(currentGroups, group)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusBadRequest, "%s", err)

		return
	}

	err = s.replaceUpstreamRuleGroups(ctx, groups)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	aghhttp.WriteJSONResponseOK(ctx, s.logger, w, r, upstreamRuleGroupForResponse(saved))
}

func (s *Server) handleUpstreamRuleGroupDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s.upstreamRuleGroupsMu.Lock()
	defer s.upstreamRuleGroupsMu.Unlock()

	req := upstreamRuleGroupIDRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusBadRequest, "decoding request: %s", err)

		return
	}

	groups := s.currentUpstreamRuleGroups()
	index := slices.IndexFunc(groups, func(group upstreamrules.Group) bool { return group.ID == req.ID })
	if index < 0 {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusNotFound, "group %d not found", req.ID)

		return
	}
	groups = slices.Delete(groups, index, index+1)
	err = s.replaceUpstreamRuleGroups(ctx, groups)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	aghhttp.OK(ctx, s.logger, w)
}

func (s *Server) handleUpstreamRuleGroupRefresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s.upstreamRuleGroupsMu.Lock()
	defer s.upstreamRuleGroupsMu.Unlock()

	req := upstreamRuleGroupIDRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusBadRequest, "decoding request: %s", err)

		return
	}

	groups := s.currentUpstreamRuleGroups()
	index := slices.IndexFunc(groups, func(group upstreamrules.Group) bool { return group.ID == req.ID })
	if index < 0 || groups[index].Kind != groupKindSubscription {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusNotFound, "subscription %d not found", req.ID)

		return
	}

	rules, fetchErr := fetchUpstreamRuleSubscription(ctx, s.conf.HTTPClient, groups[index].URL)
	if fetchErr != nil {
		groups[index].LastError = fetchErr.Error()
		s.serverLock.Lock()
		s.conf.UpstreamRuleGroups = groups
		s.serverLock.Unlock()
		s.conf.ConfModifier.Apply(ctx)
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusBadGateway, "%s", fetchErr)

		return
	}

	groups[index].Rules = rules
	groups[index].LastUpdated = time.Now().UTC().Format(time.RFC3339)
	groups[index].LastError = ""
	err = s.replaceUpstreamRuleGroups(ctx, groups)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	aghhttp.WriteJSONResponseOK(
		ctx,
		s.logger,
		w,
		r,
		upstreamRuleGroupForResponse(groups[index]),
	)
}

func (s *Server) handleUpstreamRuleTest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := upstreamRuleTestRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusBadRequest, "decoding request: %s", err)

		return
	}

	s.serverLock.RLock()
	runtime := s.upstreamRules
	s.serverLock.RUnlock()
	if runtime == nil || runtime.matcher == nil {
		aghhttp.WriteJSONResponseOK(ctx, s.logger, w, r, &upstreamRuleTestResponse{})

		return
	}

	match, ok := runtime.matcher.Match(req.Domain)
	resp := &upstreamRuleTestResponse{Matched: ok}
	if ok {
		resp.GroupID = match.GroupID
		resp.GroupName = match.GroupName
		resp.Upstreams = match.Upstreams
	}

	aghhttp.WriteJSONResponseOK(ctx, s.logger, w, r, resp)
}
