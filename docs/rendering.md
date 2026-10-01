# Render settings

[Back to README](../README.md)

## Page limit and trimming

When a resume exceeds `page_limit`, the lowest-scored bullets are trimmed until it fits.
Bullets matching higher-priority profile tags survive longer.

```toml
[render]
page_limit = 1.0
page_height_pt = 841.89   # Must match template.typ; A4 = 841.89, US Letter = 792

[render.min_elements]
job_bullets = 1
project_bullets = 1
skill_items = 1
```

A job or project with fewer included bullets than its minimum is dropped entirely.
The same applies to skill categories with fewer included items.

## Renderer backend

The shipped config selects `use_container = "auto"`. Engine probe order is Podman,
then Docker. The render image is built locally on first use and reused afterward;
its base image and Typst download may require network access during the build.

Rendering itself uses a throwaway container with `--read-only --network=none
--cap-drop=ALL --security-opt=no-new-privileges` as your UID/GID, plus
`--userns=keep-id` on Podman. On SELinux, the mounted appdir uses `:Z` relabeling.

```toml
[render]
use_container = "auto"   # Container when available, otherwise host
# use_container = "true"  # Require a working container backend
# use_container = "false" # Host Typst only
```

Under `auto`, a missing engine or failed image build falls back to the host renderer
and prints a `rendering: host (...)` banner on stderr. `true` treats backend failures
as errors. Omitting the key entirely or setting it to `""` selects host rendering.
Host mode requires `typst` on `PATH`, or a custom `[paths] typst_bin`.

## Security & hardening

Files carrying resume data are private by default: tracker entries, Markdown/filtered
TOML exports, prompt output files, and generated Typst source use `0600`; tracker
directories use `0700`. Rendered PDFs use `0644` so they can be shared.

The sanitizer is always on. Only allowed inline markup survives, and link URLs must
use an allowed scheme. Malformed markup or disallowed URLs fail the render.
`--force` emits offending content as escaped literal text instead.

Other hardening options are off unless enabled:

```toml
[render]
strip_metadata = true   # Requires qpdf on PATH
strict_input = true

[render.limits]
short = 256
bullet_text = 4096
url_or_path = 2048
```

- `strip_metadata` clears `/Author`, `/Creator`, `/Producer`, `/CreationDate`, and
  `/ModDate` through qpdf.
- NUL bytes are always rejected. `strict_input` additionally rejects control
  characters (except newline/tab), invalid UTF-8, and fields exceeding byte limits.
- Limits apply to names/titles/dates/locations/tags (`short`), bullets/summaries
  (`bullet_text`), and contact links/path-like fields (`url_or_path`).

## LLM-ready outputs

```toml
[render]
emit_markdown = true
emit_filtered = true
```

These optional siblings describe the exact resume after filtering and trimming:

- `output/<profile>.md`: Markdown grouped by job/project, projected to the profile's
  language, with your inline markup preserved as authored.
- `output/<profile>.filtered.toml`: post-filter entities with all languages intact.

Neither option changes the PDF. Paste the Markdown and a job description into your
own LLM to tailor content. resumegen does not call an LLM itself.
