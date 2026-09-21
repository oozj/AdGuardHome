# Upstream DNS rule groups

## Goal

Move the editable global upstream server field out of the DNS settings page and
add a dedicated Upstream DNS Servers page.  The new page manages the existing
default upstreams plus ordered subscription and custom rule groups.  Each group
selects its own upstream DNS servers.

## DNS semantics

Rules are evaluated against the DNS question name only.  URL schemes, paths,
queries, and fragments are discarded because clients never send them to a DNS
server.  Supported imported syntax includes AutoProxy-style domain anchors,
URLs, host wildcards, regular expressions, plain host patterns, comments, and
`@@` exceptions.  URL rules are normalized to their hostname.  Regex rules run
against the normalized QNAME and synthetic HTTP/HTTPS root URLs.  An exception
causes evaluation to continue with the next lower-priority group.

Priority is: client-specific upstream configuration, enabled rule groups in
their displayed order, then the existing default upstreams.  A group may be a
remote subscription or an editable custom group.  Invalid or hostless rules
remain visible with a compile warning and never pretend to be active.

## Backend

Add an `internal/upstreamrules` package containing the parser, compiled matcher,
subscription decoder, and immutable rule-set swap.  Persist group metadata and
cached rule text under `dns.upstream_rule_groups` in `AdGuardHome.yaml`.  Build
one shared `proxy.CustomUpstreamConfig` per enabled group during reconfigure.
Before ordinary upstream resolution, match the QNAME and select the group's
custom upstream configuration.

Expose authenticated `/control/upstream_rules/*` endpoints to list, save,
delete, refresh, edit custom text, reorder, and test a domain.  Subscription
downloads use AdGuard Home's configured HTTP client, size/time limits, atomic
validation, and retain the last good rules when an update fails.

## Frontend

Remove the Upstream component from the DNS settings page but retain bootstrap,
fallback, private reverse DNS, cache, access, and other DNS controls.  Add a
Settings submenu entry and route for Upstream DNS Servers.  The page reuses the
allowlists table, switch, refresh, add/edit/delete dialogs, and native theme.

The first row represents the existing default upstreams.  Subscription dialogs
collect name, URL, upstream servers, enabled state, and priority.  Custom groups
collect the same fields without a URL.  Their Edit Rules action opens a full
page editor modeled on Custom Rules, including validation counts and a domain
match tester that shows the selected group and upstream.

## Verification and deployment

Write parser/matcher tests before implementation, HTTP and persistence tests,
SolidJS store/component tests, typecheck/lint/unit gates, and a production
build.  Build a versioned custom Docker image, back up CT211 configuration and
current image identity, migrate the existing default upstreams and merged rule
file, then validate UI, subscriptions, custom editing, priority/exception
matching, UDP/TCP DNS, rewrites, filters, restart persistence, and rollback.
