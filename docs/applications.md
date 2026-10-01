# Applications and terminal UI

[Back to README](../README.md)

## Application tracker

Each application is a local TOML file under `applications/`, recording where you
applied, artifacts sent, contacts, notes, followups, and an append-only event log.
The tracker is offline and git-versionable; it never calls an LLM or the network.

```sh
resumegen apply new --company "Acme" --role "Senior Go Engineer" \
    --profile go-backend --jd jd/acme.md
resumegen apply set <id> status interview
resumegen apply edit <id> --company "Acme GmbH"
resumegen apply followup <id> --action "nudge recruiter"
resumegen apply followup <id> --done 1
resumegen apply contact <id> --name "Ivan" --role recruiter
resumegen apply note <id> "team uses NATS heavily"
resumegen apply list --status screen,interview --stale 14
resumegen apply show <id>
resumegen apply reopen <id>
resumegen apply delete <id> --yes
```

`new` creates a drafting entry. `set` fills in skipped stages, recording each event.
Editing fields never changes the id. `list` shows the next followup in its DUE column;
`show` numbers followups so `--done <number>` can mark one complete.

```text
drafting -> applied -> screen -> interview -> offer -> accepted
active -> withdrawn
submitted -> rejected
```

`ghosted` is applied automatically on read when a submitted application has had no
activity for `ghost_after_days`. There is no daemon; it happens next time you list or
show entries. Drafts never ghost. `ghosted` cannot be set by hand; use `withdrawn` to
close an application yourself, or `reopen` to bring a closed one back.

A JD is a text file you keep and reference with `--jd <path>`. resumegen stores the
path, never fetches it, and prompt commands can read it via `--app <id>`.

| `[tracker]` key | Default | Meaning |
|-----------------|---------|---------|
| `ghost_after_days` | 30 | Inactivity threshold for auto-ghosting submitted applications |
| `followup_default_lag_days` | 7 | Offset from today when a followup omits `--due` |

`apply list` and `apply show` support `--json`. Exit codes: `0` success,
`1` resolution error (unknown id, invalid transition, missing followup arguments),
`2` usage error (bad flag/date).

## Interactive TUI

`resumegen tui` is a front-end over the same CLI capabilities. It needs a real
terminal and does not start on a pipe. Use `1`-`6` to switch screens, `?` for the
keybind overlay, and `q` to quit.

| # | Screen | Actions |
|---|--------|---------|
| 1 | Dashboard | Status counts, ghosting risk, due followups; `n` creates an application |
| 2 | Applications | List/detail, create, status, notes, followups, events; `/` filters, `y` copies id |
| 3 | Generate | Pick a profile and render in the background; `esc` cancels |
| 4 | Prompts | Pick a template, fill inputs, run; `y` copies the result |
| 5 | Data | Open `data/*.toml` in `$EDITOR` |
| 6 | Config | Read-only effective config and appdir origin |

Every action uses the same backend as the CLI. `[tui] theme` currently accepts only
`default`; unknown values fall back to it.

`make tui` runs in the dev container with your appdir mounted. Rootless Podman keeps
files owned by you. With rootful Docker, mounted writes become root-owned. The
container lacks `wl-copy`/`xclip`, and editing uses BusyBox `vi` regardless of
`$EDITOR`. Run the host binary for clipboard support and your preferred editor.

The TUI uses [Bubble Tea](https://github.com/charmbracelet/bubbletea). Build with
`go build -tags notui ./cmd/resumegen` to omit Charm dependencies. That binary reports
that TUI support was excluded; the other commands still work.
