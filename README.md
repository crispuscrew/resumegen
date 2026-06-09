# resumegen

CLI tool that generates many PDF resumes from a single set of TOML data files via [Typst](https://typst.app).

If you have experience in different areas, you may need different resumes - each highlighting what's relevant. This tool automates that.

Inspired by [Jake's Resume](https://www.overleaf.com/latex/templates/jakes-resume/syzfjbzwjncs)

![Example resume](assets/default.png)

## Quick start

### 1. Install Typst

Download from [typst.app](https://typst.app/docs/installation/) or use your package manager.

### 2. Get resumegen

Go to the [Releases page](https://github.com/crispuscrew/resumegen/releases/latest), download the binary for your platform, and place it somewhere in your `PATH`.

### 3. First run

```sh
resumegen
```

On first launch you will be prompted to copy the default configuration to `~/.config/resumegen/`.
Prefer a self-contained, git-versionable setup? Run `resumegen init` in a directory instead — see
**Workspaces** below.

### 4. Fill in your data

Edit the five TOML files under `~/.config/resumegen/data/`: `header.toml` (name,
contacts, summary), `jobs.toml` (work experience), `projects.toml` (side projects),
`education.toml` (degrees), and `skills.toml` (skill categories).

### 5. Set up a profile

A profile defines which tags to include and in what priority order:

```toml
# ~/.config/resumegen/profiles/go-backend.toml
tags   = ["go", "backend", "devops"]
lang   = "en"
output = "go-backend.pdf"
```

### 6. Generate

```sh
resumegen --profile go-backend
# → ~/.config/resumegen/output/go-backend.pdf
```

## Usage

```sh
resumegen [--profile <name>] [--path <appdir>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--profile` | `default` | Profile name to use (matches `profiles/<name>.toml`) |
| `--path` | walk-up, then `~/.config/resumegen` | Path to the application directory (see [Workspaces](#workspaces)) |
| `--lang` | from profile | Override the output language |
| `--force` | off | Render even if a bullet has malformed markup or a disallowed URL — the sanitizer falls back to literal text (see [Security & hardening](#security--hardening)) |
| `--version` | — | Print version and exit |

Subcommands: `resumegen init` (bootstrap a workspace), `resumegen template extract`, `resumegen prompt extract`.

Output PDF is written to `<appdir>/output/<profile.output>`.

## App directory structure

```
~/.config/resumegen/
├── config.toml          # Paths and render settings
├── profiles/            # One .toml per target role
│   ├── default.toml
│   └── cpp-embedded.toml
├── data/                # Resume content
│   ├── header.toml
│   ├── jobs.toml
│   ├── projects.toml
│   ├── education.toml
│   └── skills.toml
└── templates/           # Typst templates (auto-managed)
```

## Workspaces

Instead of the single global `~/.config/resumegen/`, you can keep a self-contained,
git-versionable appdir anywhere — useful for tracking your resume data alongside other
projects, or keeping separate sets.

```sh
resumegen init my-resume      # creates my-resume/ with a .resumegen/ marker + example data
cd my-resume
resumegen --profile default   # discovered automatically by walking up from the CWD
```

`init` writes a `.resumegen/` marker that enables **walk-up discovery**: run `resumegen`
from anywhere inside the tree and it finds the workspace. Resolution order is `--path` >
nearest `.resumegen/` marker above the CWD > the default `~/.config/resumegen/`. A workspace
`config.toml` is layered on top of the global one when both exist (workspace wins).

| `init` flag | Effect |
|-------------|--------|
| (default) / `--with-example` | Marker + example data and profiles |
| `--bare` | Marker only; no data |
| `--full-example` | Also extract templates and prompts for editing |
| `--name`, `--description` | Metadata written into the marker |

`init` is idempotent and never overwrites existing files. To override a bundled default
selectively, use `resumegen template extract [name...]` or `resumegen prompt extract <name>` —
these copy the embedded default into your appdir, where it then shadows the built-in.

## Profiles

A profile selects and ranks content by tags. Tags are ordered highest to lowest priority - they drive both filtering and trim order when the resume exceeds the page limit.

```toml
# profiles/go-backend.toml
tags   = ["go", "backend", "devops"]
lang   = "en"   # any language key present in your data; always falls back to "en"
output = "go-backend.pdf"
```

Jobs and projects with no matching tags are excluded entirely. Bullets with no matching tags are dropped. If a job ends up with no visible bullets, it is dropped too.

## Data files

Content fields support Typst inline markup: `*bold*`, `_italic_`, `#link("url")[text]`.

### jobs.toml

Job-level `tags` control whether the entire position is shown. Bullet-level `tags` control individual bullet visibility. If a job has no top-level tags, its visibility is determined by its bullets alone.

```toml
[[jobs]]
    tags    = ["go", "backend", "devops"]   # job hidden entirely if none match
    company = "Acme Corp"

    [jobs.title]
    en = "Software Engineer"
    ru = "Инженер-программист"

    [jobs.date]
    en = "Jan. 2025 – Present"

    [jobs.location]
    en = "Moscow, Russia"

    [[jobs.bullets]]
    tags = ["go", "backend"]
    [jobs.bullets.text]
    en = "Built a *REST API* service in Go"
    ru = "Разработал сервис *REST API* на Go"
```

> **Note:** Flat fields like `company` must appear **before** the first `[jobs.*]` subtable in each entry, otherwise TOML will assign them to the wrong table.

### skills.toml

```toml
[[categories]]

    [categories.name]
    en = "Languages"
    ru = "Языки программирования"

    [[categories.items]]
    name = "Go"
    tags = ["go", "backend"]

    [[categories.items]]
    name = "C/C++"
    tags = ["cpp", "embedded"]
```

### education.toml

Education entries are always shown in full - no tag filtering.

```toml
[[edu]]

    [edu.title]
    en = "Moscow State University"

    [edu.degree]
    en = "B.S. Computer Science"

    [edu.location]
    en = "Moscow, Russia"

    [edu.date]
    en = "2020 – 2024"
```

## Page limit and trimming

When a resume exceeds `page_limit`, the tool automatically trims the lowest-scored bullets until the resume fits. Scoring is based on tag priority: bullets matching higher-priority profile tags score higher and are kept longer.

You can tune the behavior in `config.toml`:

```toml
[render]
page_limit     = 1.0      # trim until the resume fits this many pages
page_height_pt = 841.89   # must match the paper size in template.typ (A4 = 841.89, US Letter = 792)

[render.min_elements]
job_bullets     = 1   # a job with fewer included bullets than this is dropped entirely
project_bullets = 1   # same for projects
skill_items     = 1   # a skill category with fewer included items than this is dropped entirely
```

## Security & hardening

resumegen takes a careful stance toward the data it renders and the PDFs it produces.
All of the following except the sanitizer are **opt-in and off by default**, so v1.0
output is unchanged unless you enable them in `[render]`.

### Markup sanitizer (always on)

Content fields are passed through a Typst sanitizer before rendering. Only an allowlist of
inline markup survives — `*bold*`, `_italic_`, raw/code spans, and links — and link URLs are
validated against an allowed-scheme list. A bullet with malformed markup or a disallowed URL
fails the render by default. Pass `--force` to render anyway: the offending content is emitted
as Typst-escaped literal text instead of being interpreted.

### Containerized render — `use_container`

Run Typst inside a throwaway rootless container instead of the host binary. The engine probe
order is podman, then docker, and the container runs locked down:
`--read-only --network=none --cap-drop=ALL --security-opt=no-new-privileges` as your own UID/GID
(plus `--userns=keep-id` on podman).

```toml
[render]
use_container = "auto"   # ""/"false" = host (default) · "true" = require an engine · "auto" = engine if present, else host
```

With `"auto"`, a missing engine or failed image build falls back to the host renderer and prints
a one-line `rendering: host (...)` banner on stderr. Host mode is byte-identical to v1.0.

### PDF metadata stripping — `strip_metadata`

After rendering, rebuild the PDF through `qpdf` to empty its `/Author`, `/Creator`, `/Producer`,
`/CreationDate`, and `/ModDate`. Requires `qpdf` on `PATH`.

```toml
[render]
strip_metadata = true
```

### Strict input validation — `strict_input`

NUL bytes in your data are **always** rejected. Turning on `strict_input` additionally rejects
control characters (except newline and tab), invalid UTF-8, and fields that exceed per-class byte
limits — catching corrupt or hostile data before it reaches the renderer.

```toml
[render]
strict_input = true

[render.limits]      # optional; these are the defaults
short       = 256    # names, titles, dates, company, location, tags
bullet_text = 4096   # bullet text and the header summary
url_or_path = 2048   # contact hrefs and path-like fields
```

## Build from source

### Requirements

- [Go 1.24+](https://go.dev/dl/)
- [Podman](https://podman.io) or [Docker](https://docker.com) (for my make commands)

### Build

```sh
make build
```

Binary is placed at `./bin/resumegen`.

## Development

```sh
make lint     # run golangci-lint
make test     # run tests
make tidy     # go mod tidy
make rebuild  # force rebuild all container images
make clean    # remove build artifacts
```

## Plans

- Transfer from typst to smth embedded in this binary
- Automated tests
- Chronological or manual ordering of bullets and entries
- Verbose mode for debugging filter and trim decisions

## P.S.

If this project helped u find a job - show your new employer [a resume generated from the defaults](assets/default.pdf) :)