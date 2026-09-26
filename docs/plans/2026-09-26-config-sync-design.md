# Configuration synchronization

## Goal

Add a Configuration Sync page under Settings.  A node can be unconfigured, a
primary server, or a secondary server.  A secondary server exposes a copyable
pairing link.  A primary server stores one or more pairing links, pushes its
configuration immediately when a secondary is added, and pushes again after
later configuration writes.

## Synchronization scope

The payload is the complete `AdGuardHome.yaml` configuration except for the
top-level `config_sync` section.  The secondary keeps its own role, pairing
token, and peer state.  Runtime data such as query logs, statistics databases,
sessions, downloaded filter cache files, and process-specific files are not
copied.

The primary at `192.168.50.60` is authoritative.  `192.168.50.1` and
`192.168.50.2` are secondary servers.  All three nodes must run the same custom
version before the primary performs the first synchronization.

## Pairing and transport

A secondary creates a cryptographically-random token and presents a link whose
HTTP fragment contains the token.  The fragment is not sent by browsers or
ordinary HTTP clients.  The primary parses the link, stores the endpoint and
token in its local `config_sync` section, and authenticates pushes with a
Bearer header.  Pairing links support HTTP and HTTPS; HTTPS is preferred when
the nodes expose it.

Authenticated administrator endpoints manage role, peers, token regeneration,
status, and manual synchronization.  The receive endpoint bypasses web-session
authentication but accepts only a valid secondary token and a bounded request
body.

## Safe application

The secondary merges its local `config_sync` section into the received YAML,
writes a temporary file, validates it with the current AdGuard Home binary and
`--check-config`, and atomically replaces the live file only after validation.
It replies before requesting a supervised restart.  Invalid payloads and failed
validation leave the current configuration untouched.

Primary pushes are debounced and serialized.  One unavailable secondary does
not block other peers.  The page reports the most recent in-process result and
offers a manual retry.

## Interface

The page uses existing AdGuard Home cards, controls, tables, buttons, toasts,
and responsive layout.  Primary mode shows the peer list, an add-link field,
remove actions, last result, and Sync now.  Secondary mode shows the generated
link, Copy, and Regenerate link.  Switching roles requires an explicit Save.

## Verification and rollout

Write parser, merge, token, authentication, payload-limit, validation-failure,
atomic-write, and push tests before implementation.  Add frontend helper and
component tests, then run Go lint/tests/race tests and frontend lint, typecheck,
tests, and production build.

Deploy a visible trial to `192.168.50.2` first.  After interface acceptance,
update `192.168.50.1` and `192.168.50.60`, configure `.1/.2` as secondary,
configure `.60` as primary, add both links, and verify configuration checksums,
DNS over UDP/TCP, web management, rule groups, subscription refresh, and
restart recovery on all nodes.
