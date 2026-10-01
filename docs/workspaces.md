# CLI and workspaces

[Back to README](../README.md)

```sh
resumegen [--profile <name>] [--path <appdir>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--profile` | `default` | Profile matching `profiles/<name>.toml` |
| `--path` | walk-up, then `~/.config/resumegen` | Application directory |
| `--lang` | from profile | Override output language |
| `--force` | off | Emit malformed markup as literal text instead of failing |
| `--version` | - | Print version and exit |

Run `resumegen help` for the full command list. Output lands at
`<appdir>/output/<profile.output>`. A relative `output` may step outside `output/`
(for example, `../drafts/x.pdf`) but never outside the appdir itself.

## App directory structure

```text
~/.config/resumegen/
  config.toml       # Paths and render settings; all keys optional
  profiles/         # One TOML file per target role
  data/             # header, jobs, projects, education, skills
  templates/        # Typst layout templates
  prompts/          # Prompt templates
  applications/     # Job-application tracker entries
  output/           # PDFs and optional Markdown/TOML dumps
```

## Workspaces

Keep a self-contained, git-versionable appdir anywhere instead of using the global one:

```sh
resumegen init my-resume
cd my-resume
resumegen --profile default
```

The `.resumegen/` marker enables discovery by walking up from the current directory.
Resolution order is `--path` > nearest marker above the CWD > `~/.config/resumegen/`.
Workspace `config.toml` layers over global config; workspace keys win.

| `init` flag | Effect |
|-------------|--------|
| (default) / `--with-example` | Marker + example data and profiles |
| `--bare` | Marker only; no data |
| `--full-example` | Also extract templates and prompts for editing |
| `--name`, `--description` | Metadata written into the marker |

`init` is idempotent and never overwrites existing files. Extract a bundled template
selectively with `resumegen template extract [name...]` or
`resumegen prompt extract <name>`. Your appdir copy shadows the built-in.

## Profiles

```toml
# profiles/go-backend.toml
tags   = ["go", "backend", "devops"]
lang   = "en"
output = "go-backend.pdf"
```

Tags are ordered highest to lowest priority. Jobs and projects with no matching tags
are excluded, as are bullets with no matching tags. A job with no top-level tags is
judged by its bullets. Jobs left without visible bullets and skill categories left
without visible items are dropped. Education is not filtered.
