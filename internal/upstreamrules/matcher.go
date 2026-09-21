// Package upstreamrules matches DNS question names to ordered upstream groups.
package upstreamrules

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// Group is one ordered upstream rule group.
type Group struct {
	ID          uint64   `json:"id"                    yaml:"id"`
	Name        string   `json:"name"                  yaml:"name"`
	Enabled     bool     `json:"enabled"               yaml:"enabled"`
	Priority    int      `json:"priority"              yaml:"priority"`
	Kind        string   `json:"kind"                  yaml:"kind"`
	URL         string   `json:"url,omitempty"         yaml:"url,omitempty"`
	Upstreams   []string `json:"upstreams"             yaml:"upstreams"`
	Rules       string   `json:"rules,omitempty"       yaml:"rules,omitempty"`
	RulesCount  int      `json:"rules_count,omitempty" yaml:"-"`
	LastUpdated string   `json:"last_updated,omitempty" yaml:"last_updated,omitempty"`
	LastError   string   `json:"last_error,omitempty"   yaml:"last_error,omitempty"`
}

// Issue describes a rule that could not be compiled.
type Issue struct {
	GroupID   uint64 `json:"group_id"`
	GroupName string `json:"group_name"`
	Line      int    `json:"line"`
	Message   string `json:"message"`
}

// Match is the selected group for a DNS question name.
type Match struct {
	GroupID   uint64
	GroupName string
	Upstreams []string
}

// Matcher matches DNS question names to upstream groups.
type Matcher struct {
	groups []compiledGroup
}

type ruleKind uint8

const (
	ruleKindAll ruleKind = iota + 1
	ruleKindSuffix
	ruleKindContains
	ruleKindRegexp
)

type compiledRule struct {
	re   *regexp.Regexp
	text string
	kind ruleKind
}

type compiledGroup struct {
	group      Group
	positive   []compiledRule
	exceptions []compiledRule
}

// Compile compiles groups into a matcher.
func Compile(groups []Group) (m *Matcher, issues []Issue) {
	groups = slices.Clone(groups)
	slices.SortStableFunc(groups, func(left, right Group) int {
		switch {
		case left.Priority < right.Priority:
			return -1
		case left.Priority > right.Priority:
			return 1
		default:
			return 0
		}
	})

	m = &Matcher{}
	for _, group := range groups {
		if !group.Enabled || len(group.Upstreams) == 0 {
			continue
		}

		compiled, groupIssues := compileGroup(group)
		issues = append(issues, groupIssues...)

		if len(compiled.positive) > 0 {
			m.groups = append(m.groups, compiled)
		}
	}

	return m, issues
}

func compileGroup(group Group) (compiled compiledGroup, issues []Issue) {
	compiled = compiledGroup{group: group}
	for lineIndex, line := range strings.Split(group.Rules, "\n") {
		rule, exception, ignore, err := compileLine(line)
		if ignore {
			continue
		}
		if err != nil {
			issues = append(issues, Issue{
				GroupID:   group.ID,
				GroupName: group.Name,
				Line:      lineIndex + 1,
				Message:   err.Error(),
			})

			continue
		}

		if exception {
			compiled.exceptions = append(compiled.exceptions, rule)
		} else {
			compiled.positive = append(compiled.positive, rule)
		}
	}

	return compiled, issues
}

// Match returns the first matching group.
func (m *Matcher) Match(qname string) (match Match, ok bool) {
	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(qname)), ".")
	if host == "" {
		return Match{}, false
	}

	for _, group := range m.groups {
		if matchesAny(group.exceptions, host) || !matchesAny(group.positive, host) {
			continue
		}

		return Match{
			GroupID:   group.group.ID,
			GroupName: group.group.Name,
			Upstreams: slices.Clone(group.group.Upstreams),
		}, true
	}

	return Match{}, false
}

// DecodeSubscription decodes a remote subscription body.
func DecodeSubscription(body []byte) (rules string, err error) {
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return "", nil
	}

	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
	} {
		decoded, decErr := encoding.DecodeString(raw)
		if decErr == nil && utf8.Valid(decoded) && looksLikeRuleList(string(decoded)) {
			return string(decoded), nil
		}
	}

	if !utf8.Valid(body) {
		return "", fmt.Errorf("subscription is not valid UTF-8")
	}

	return string(body), nil
}

func looksLikeRuleList(s string) bool {
	return strings.Contains(s, "\n") && (strings.Contains(s, "||") ||
		strings.Contains(s, "[AutoProxy") || strings.Contains(s, "@@"))
}

func matchesAny(rules []compiledRule, host string) bool {
	for _, rule := range rules {
		if rule.match(host) {
			return true
		}
	}

	return false
}

func (r compiledRule) match(host string) bool {
	switch r.kind {
	case ruleKindAll:
		return true
	case ruleKindSuffix:
		return host == r.text || strings.HasSuffix(host, "."+r.text)
	case ruleKindContains:
		return strings.Contains(host, r.text)
	case ruleKindRegexp:
		return r.re.MatchString(host) ||
			r.re.MatchString("http://"+host+"/") ||
			r.re.MatchString("https://"+host+"/")
	default:
		return false
	}
}

func compileLine(line string) (rule compiledRule, exception, ignore bool, err error) {
	line = strings.TrimSpace(line)
	if isIgnoredLine(line) {
		return compiledRule{}, false, true, nil
	}

	if exception = strings.HasPrefix(line, "@@"); exception {
		line = strings.TrimPrefix(line, "@@")
	}

	if strings.HasPrefix(line, "/") {
		rule, err = compileRegexpLine(line)

		return rule, exception, false, err
	}

	line, _, _ = strings.Cut(line, "$")
	if line == "*" {
		return compiledRule{kind: ruleKindAll}, exception, false, nil
	}

	if strings.HasPrefix(line, "||") {
		rule, err = compileDomainAnchor(line)

		return rule, exception, false, err
	}

	line = strings.TrimPrefix(line, "|")
	if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
		rule, err = compileURLRule(line)

		return rule, exception, false, err
	}

	rule, err = compilePlainRule(line)

	return rule, exception, false, err
}

func isIgnoredLine(line string) (ignored bool) {
	return line == "" || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "#") ||
		(strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"))
}

func compileRegexpLine(line string) (rule compiledRule, err error) {
	pattern := line[1:]
	if strings.HasSuffix(pattern, "/") && !strings.HasSuffix(pattern, `\/`) {
		pattern = strings.TrimSuffix(pattern, "/")
	}
	pattern = strings.ReplaceAll(pattern, `\/`, "/")
	re, err := compileRuleRegexp(pattern)
	if err != nil {
		return compiledRule{}, fmt.Errorf("compiling regular expression: %w", err)
	}

	return compiledRule{kind: ruleKindRegexp, re: re}, nil
}

func compileDomainAnchor(line string) (rule compiledRule, err error) {
	pattern := line[2:]
	hasTerminator := false
	if idx := strings.IndexAny(pattern, "^/|"); idx >= 0 {
		hasTerminator = true
		pattern = pattern[:idx]
	}
	pattern = strings.ToLower(strings.TrimSuffix(pattern, "."))
	if pattern == "" {
		return compiledRule{}, fmt.Errorf("empty domain anchor")
	}
	if strings.Contains(pattern, "*") {
		return compileGlobRule(pattern, true)
	}
	if strings.Contains(pattern, ".") {
		return compiledRule{kind: ruleKindSuffix, text: pattern}, nil
	}

	rePattern := `(?:^|\.)` + regexp.QuoteMeta(pattern)
	if hasTerminator {
		rePattern += `(?:$|\.)`
	}

	return compiledRule{kind: ruleKindRegexp, re: regexp.MustCompile(rePattern)}, nil
}

func compileURLRule(line string) (rule compiledRule, err error) {
	host, err := hostFromURL(line)
	if err != nil {
		return compiledRule{}, err
	}
	if strings.Contains(host, "*") {
		return compileGlobRule(host, false)
	}

	return compiledRule{kind: ruleKindSuffix, text: host}, nil
}

func compilePlainRule(line string) (rule compiledRule, err error) {
	plain := strings.ToLower(strings.Trim(line, "|^"))
	if plain == "" || strings.ContainsAny(plain, " /[]") {
		return compiledRule{}, fmt.Errorf("rule has no executable hostname semantics")
	}
	if strings.Contains(plain, "*") {
		return compileGlobRule(plain, false)
	}
	if strings.Contains(plain, ".") {
		return compiledRule{kind: ruleKindSuffix, text: plain}, nil
	}

	return compiledRule{kind: ruleKindContains, text: plain}, nil
}

func compileGlobRule(pattern string, anchoredAtBoundary bool) (rule compiledRule, err error) {
	re, err := compileHostGlob(pattern, anchoredAtBoundary)
	if err != nil {
		return compiledRule{}, err
	}

	return compiledRule{kind: ruleKindRegexp, re: re}, nil
}

var hostnameSuffixRegexp = regexp.MustCompile(
	`([a-z0-9-]+(?:\\\.[a-z0-9-]+)+)\$$`,
)

func compileRuleRegexp(pattern string) (re *regexp.Regexp, err error) {
	re, err = regexp.Compile(pattern)
	if err == nil {
		return re, nil
	}

	lookaheadStart := strings.Index(pattern, `(?=.*?(`)
	if lookaheadStart < 0 {
		return nil, err
	}
	tokensStart := lookaheadStart + len(`(?=.*?(`)
	tokensEnd := strings.Index(pattern[tokensStart:], `))`)
	if tokensEnd < 0 {
		return nil, err
	}
	tokens := pattern[tokensStart : tokensStart+tokensEnd]
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9|-]+$`, tokens); !matched {
		return nil, err
	}

	suffixMatch := hostnameSuffixRegexp.FindStringSubmatch(pattern)
	if len(suffixMatch) != 2 {
		return nil, err
	}
	suffix := strings.ReplaceAll(suffixMatch[1], `\.`, ".")
	hostPattern := `^[a-z0-9.-]*(?:` + tokens + `)[a-z0-9.-]*\.` +
		regexp.QuoteMeta(suffix) + `$`

	return regexp.Compile(hostPattern)
}

func hostFromURL(ruleURL string) (host string, err error) {
	u, err := url.Parse(ruleURL)
	if err != nil {
		return "", fmt.Errorf("parsing URL rule: %w", err)
	}

	host = strings.ToLower(u.Hostname())
	if host == "" {
		return "", fmt.Errorf("URL rule has no hostname")
	}

	return host, nil
}

func compileHostGlob(pattern string, anchoredAtBoundary bool) (re *regexp.Regexp, err error) {
	quoted := regexp.QuoteMeta(strings.ToLower(pattern))
	quoted = strings.ReplaceAll(quoted, `\*`, `[^.]*`)
	if anchoredAtBoundary {
		quoted = `(?:^|\.)` + quoted
	} else {
		quoted = `^` + quoted
	}
	re, err = regexp.Compile(quoted + `$`)
	if err != nil {
		return nil, fmt.Errorf("compiling hostname glob: %w", err)
	}

	return re, nil
}
