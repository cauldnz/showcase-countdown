# Plan: agent-driven sticks for the Hack Showcase

Status: draft for review, 2026-09-08. Event: Thursday 17 September 2026, 2pm.

## 1. Purpose

About a dozen M5StickC Plus SE units sit on team tables at a coding event where
teams learn agentic software engineering for enterprise apps. The sticks count
down to the showcase and play a distributed fanfare at 2pm. This plan adds a
**messaging layer** so each team can drive its own stick from its coding
agents through an MCP server: show messages, play audio, flash the LED, shout to
the room, and message other teams.

**Priority.** The countdown and fanfare are the protected core. Messaging is a
best-effort side-show. The archetypal use is "play a jingle every time someone
merges a PR". Nothing in the event depends on messaging working, and nothing in
messaging may break the fanfare.

## 2. Decisions

| # | Decision | Choice | Notes |
| --- | --- | --- | --- |
| 1 | Purpose | Side-show, fanfare protected | Teams may wire agents to their stick for fun |
| 2 | Binding | Claim code shown on the stick | No device-to-team spreadsheet |
| 3 | Identity | The claim code *is* the credential | No tokens, no registration. Whole table shares it |
| 4 | Code visibility | Always on screen | Mischief between tables is accepted |
| 5 | Transport | MCP over streamable HTTP only | No webhook API, no stdio shim, no direct MQTT for teams |
| 6 | Reachability | **To be determined** | Default assumption: laptops join the AX3000's WiFi |
| 7 | Room lock | Auto-lock T-60s to fanfare end, plus organiser manual lock | Enforced in server and on device |
| 8 | ESP-NOW bridge | Fire signal and lock/unlock only | Spare stick on the router USB |
| 9 | Audio | Anything goes | Organiser mute is the brake. Richer audio in issue #2 |
| 10 | Social limits | Light cooldowns | Shout 1 per 120 s per team, DM 1 per 5 s, text 120 chars |
| 11 | Two-way | Button events and per-team inbox | `wait_for_button`, `inbox` tools |
| 12 | Router OS | Stay on GL.iNet firmware | Vanilla OpenWrt 24.10 only if the USB-serial kmod fails |
| 13 | Repo layout | Everything in this repo | `server/` for Go, `bridge/` for the ESP-NOW sketch |
| 14 | Dashboard | Full | Fleet grid, live log, send-from-browser, organiser controls |
| 15 | Organiser auth | Dashboard open, organiser tools need a secret | Secret from the router's env, kept in Infisical |
| 16 | Fake fleet | Yes, with simulated screens | N virtual sticks on MQTT, screens drawn in the dashboard |
| 17 | Idle screen | Code and team name in a strip under the countdown | Plus an MQTT connection dot |
| 18 | Tonight | Real stick shows a shout via the router-hosted stack | Fake fleet and dashboard demoed on the NUC |
| 19 | LVGL | Phase 2, after tonight | One UI codebase for stick and SDL simulator |

## 3. Architecture

```
team agents (Claude Code, Copilot, Cursor)  x  many people per table
        |  MCP over streamable HTTP; claim code identifies the team
        v
Go MCP server + dashboard  (GL-MT3000, port 8080)
        |  MQTT                                 |  serial /dev/ttyUSB0
        v                                       v
Mosquitto (GL-MT3000:1883)              ESP-NOW bridge stick (router USB)
        |  MQTT over 2.4 GHz WiFi                |  ESP-NOW, same channel
        v                                       v
12 x M5StickC Plus SE  <--------------------------
```

Principles:

- **Devices are dumb.** A stick understands display, audio, LED, config, state,
  and button events. Shouts and DMs are display commands with a `from` field.
- **The server owns policy.** Claim codes, cooldowns, lock, mute, inbox, and
  the organiser secret live in one Go process.
- **The router hosts what the room depends on.** Broker, NTP, server,
  dashboard, bridge. The NUC is for development only.
- **The clock is the fanfare's sync.** ESP-NOW is a backup path for the fire
  signal and the lock, not a replacement.

## 4. Components

### 4.1 Firmware (`src/`)

- WiFi stays up after NTP sync when `MQTT_HOST` is set. `WiFi.setSleep(false)`
  for delivery latency; sticks are on USB power at the event.
- `messaging.cpp`: ESP-IDF MQTT client in its own task, inbound commands
  queued to the loop task. Retained state with LWT. Subscribes to
  `showcase/dev/<id>/cmd/#` and `showcase/all/cmd/#`.
- Device id: last three bytes of the eFuse MAC, lower-case hex.
- Claim code: four digits derived from the device id and a per-build salt, so
  it is stable across reboots and not guessable from the id alone. Shown in the
  idle strip.
- Overlay layer above the countdown: header by kind (`self`, `dm`, `shout`,
  `organiser`), text with the existing colour markup, marquee if wide, TTL,
  priority. Falls back to the countdown.
- Audio: non-blocking note sequencer for composed strings; named jingles.
  Uncapped by the device. The celebration path stays blocking and wins.
- LED: colour, `solid` / `blink` / `breathe` / `off`, TTL, then back to the
  daily schedule.
- Lock: retained `config.locked` and an ESP-NOW lock frame both make the
  device ignore non-organiser commands. Auto-lock also computed locally from
  the event epoch, so a stick with no network still protects the fanfare.
- ESP-NOW receiver on the AP's channel for `fire` and `lock` frames. `fire`
  starts the celebration only if within a few seconds of the target, so a
  stray frame cannot trigger it early.
- Button A/B presses publish events. Existing behaviours (resync, lights
  toggle, diagnostics) stay on the same buttons.

### 4.2 MCP server and dashboard (`server/`)

- Go, official MCP Go SDK, Paho MQTT, `net/http`. One static `linux/arm64`
  binary, also runs on Windows for development. procd service on the router.
- MCP endpoint `POST /mcp`. The claim code comes from an `X-Claim-Code`
  header or a `?code=` query param. Organiser tools need `X-Organiser-Secret`.
- Tools, team scope: `claim(code, team_name)`, `status()`, `show(text,
  seconds)`, `play(jingle | notes)`, `led(color, mode, seconds)`,
  `shout(text)`, `message_team(team, text)`, `list_teams()`, `inbox()`,
  `wait_for_button(timeout_s)`.
- Tools, organiser scope: `broadcast`, `lock`, `unlock`, `mute(team)`,
  `unmute`, `rename`, `unbind`.
- Policy: cooldowns per decision 10, lock per decision 7, mute list, inbox
  ring buffer per team, audit log.
- Dashboard at `/`: fleet grid with a 240x135 canvas per stick drawn from the
  same JSON the devices receive, real sticks alongside virtual ones, online and
  battery state, last message, live event log over WebSocket, send-from-browser
  for any stick, and organiser controls that prompt for the secret.
- Fake fleet: `server --fake N` spawns N virtual sticks as MQTT clients with
  real-looking ids, state, and button buttons in the dashboard.

### 4.3 Router (`docs/router.md`, issue #1)

- Cable on the WAN port. 2.4 GHz SSID and key match the sticks' `.env`.
  2.4 GHz channel pinned for ESP-NOW.
- Mosquitto via opkg, anonymous on the LAN. NTP served to the LAN by sysntpd.
  Sticks use 192.168.8.1 as their first NTP server.
- Go binary and dashboard as a procd service, organiser secret in its env.
- `kmod-usb-serial-ftdi` for the bridge stick. Fallback: vanilla OpenWrt.

### 4.4 ESP-NOW bridge (`bridge/`)

- A stick flashed with a small sketch: reads line-delimited commands on USB
  serial from the server, broadcasts ESP-NOW frames on the pinned channel.
  Frames: `lock`, `unlock`, `fire <epoch>`.

### 4.5 Secrets and config

- `.env` for the sticks comes from Infisical project `showcase-countdown`,
  secret `DOTENV`, via `scripts/fetch_env.ps1`. It gains `MQTT_HOST`,
  `MQTT_PORT`, `CLAIM_SALT`, and `ESPNOW_CHANNEL`.
- Server config: broker address, organiser secret, event epoch, fake fleet
  size. From env vars; the organiser secret is stored in Infisical too.

## 5. Open items

- **Reachability at the venue (decision 6).** Options: laptops join the
  AX3000's WiFi; router on the venue LAN; public tunnel. Affects whether the
  server binds to LAN only and whether teams need any network instructions.
- **Venue internet.** If absent, the router still serves NTP from its RTC-less
  clock, so it must be synced before travel or from a phone hotspot on arrival.
- **Event SSID.** The sticks' `.env` currently names `Countdown`. The router
  will be configured to match.
- **PR-merge jingle path.** With MCP-only transport, a CI job cannot call the
  server directly. A team's coding agent can, via a hook. Revisit a webhook API
  if teams ask.

## 6. Schedule

### Today, on the NUC

1. ~~Firmware: WiFi stays up, MQTT connect, retained state, LWT, display overlay,
   claim strip. Test with Mosquitto in Podman.~~ **Done.**
2. ~~Firmware: audio sequencer and jingles, LED modes, button events, config and
   lock handling.~~ **Done.** ESP-NOW receiver still to do.
3. ~~Server: MCP endpoint, claim, own-device tools, shout, DM, inbox, lock,
   cooldowns, organiser tools. Fake fleet. Dashboard with canvas screens and
   live log.~~ **Done.**
4. End-to-end on the NUC: one real stick plus eleven fake ones driven over MCP
   from a scripted client. **Done.** Claude Code as the client still to try.

### This evening, on the AX3000

5. Router audit, Mosquitto, NTP, SSID and channel, Go binary as a service.
6. Real stick joins the router, syncs, and shows a shout sent from Claude Code.
7. If the kmod installs: flash the bridge sketch and prove lock over ESP-NOW.

### Before the event

8. LVGL migration with SDL simulator (phase 2).
9. Flash a dozen, run `scripts/fleet_voices.py`, full rehearsal with lock and
   fire, including a network-loss drill.
10. One-page team handout: URL, header, tool list, three example prompts.

## 7. Risks

| Risk | Mitigation |
| --- | --- |
| RAM: WiFi plus MQTT plus 64 KB sprite | Measure heap tonight; sprite falls back to direct draw; reduce MQTT buffer |
| Team audio storm during the day | Organiser mute, and the lock at T-60s |
| GL.iNet kmod mismatch blocks the bridge | Vanilla OpenWrt fallback, or run the bridge from the NUC's USB for the rehearsal |
| Venue WiFi hostile to a dozen sticks | Sticks and laptops on our own AP; ESP-NOW carries lock and fire |
| Agent loop floods the room | Cooldowns in the server; device drops queued commands rather than blocking |
| Fanfare stepped on | Lock enforced in server, on device from the clock, and over ESP-NOW |
