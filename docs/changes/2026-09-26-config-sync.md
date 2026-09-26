# 2026-09-26 Configuration sync

## Requirements

- Add primary and secondary roles under Settings.
- A secondary displays a pairing link.
- A primary accepts secondary links and automatically synchronizes all
  configuration except synchronization settings.
- Publish a new public version and update `192.168.50.60`, `192.168.50.1`, and
  `192.168.50.2` with `.60` as the authoritative primary.

## Current stage

- Design, backend, and classic frontend implemented.
- Live HTTP reachability confirmed on all three nodes.
- `.1` is the custom RouterOS container and `.2` is the native CT deployment.
- Trial `v0.107.79-custom.2` is deployed on `.2`.
- Selecting the primary role immediately reveals the secondary-link input;
  selecting the secondary role immediately reveals the pairing-link panel.
- Local Go tests, race tests, Go lint, frontend lint, typecheck, 45 tests, and
  the production build passed before the latest UI feedback; targeted tests,
  lint, typecheck, production build, and Windows compilation passed afterward.
- Trial `v0.107.79-custom.2` is deployed inside the existing `.60` container;
  its mounted configuration checksum remained unchanged during the update.
- `.60` is primary and `.2` is secondary.  The first manual attempt raced the
  `.2` role-change restart and returned connection refused; retry returned 200.
- Canonical configuration hashes excluding `config_sync` match exactly on
  `.60` and `.2`.  UDP/TCP DNS and the management interfaces passed on both.
- Continuous-save testing exposed a secondary restart-window race.  Per-peer
  concurrent retries were added and covered by regression tests.

## Remaining work

- User accepted the `.2` trial and primary/secondary behavior.
- OpenAPI and public documentation updated.
- Fresh full verification and final code review.
- GitHub publication, `.1` deployment, final three-node synchronization, and
  removal of temporary trial rollback binaries.
