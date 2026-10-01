# Prompt templates

[Back to README](../README.md)

Templates combine resume data and a job description into ready-to-paste text.
resumegen assembles the prompt; it never calls an LLM.

```sh
resumegen prompt list
resumegen prompt show tailor-bullets
resumegen prompt run tailor-bullets --jd job.txt --copy
```

A template is a Markdown file with TOML frontmatter (`prompts/<name>.md`). Its
`{{placeholders}}` are filled from declared input sources:

| Source | Fills from |
|--------|-----------|
| `data-dump` | `output/<profile>.md`; enable `emit_markdown` and render first |
| `file` | A file passed via the declared flag, such as `--resume <path>` |
| `jd-file` | A job-description file passed with `--jd <path>` |
| `flag` | A named flag, such as `--company`, `--role`, `--tone` |
| `prompt` | Interactive entry; skipped under `--no-input` |
| `stdin` | Piped input |
| `app-id` | A tracked application field via `--app <id>`; `field = "jd"` reads its JD file |

With `--app <id>`, empty `company`/`role` flags and empty `jd-file` inputs inherit
the tracked application's values. Explicit flags always win. Generic `file` inputs
have no application fallback: `import-resume` always needs its own source resume.

Bundled: `import-resume`, `analyze-jd`, `tailor-bullets`, `cover-letter`, `gap-report`,
`interview-prep`, `recruiter-reply`, `salary-research`, `followup`, `rejection-analysis`.

Copy one with `resumegen prompt extract <name>`. Your copy shadows the built-in and
is never overwritten. Bundled prompts are English, but you can add a prompt in any
language: the same filename shadows a bundled prompt; a new name adds one.

## Importing an existing resume

`import-resume` asks your LLM to convert an existing plain-text resume into five tagged
TOML data files. If the source is a PDF, extract its text first:

```sh
pdftotext my-resume.pdf my-resume.txt
resumegen prompt run import-resume --resume my-resume.txt --copy
```

Paste the assembled prompt into your LLM. Save its five fenced blocks as
`data/header.toml`, `data/jobs.toml`, `data/projects.toml`, `data/education.toml`, and
`data/skills.toml`. It also reports the tag vocabulary and anything it could not place.

| Flag | Effect |
|------|--------|
| `--resume <path>` | Required plain-text resume |
| `--tags "go, backend, ..."` | Tag vocabulary; omit to derive one from the resume |
| `--langs "en, ru"` | Languages to emit; defaults to `en` |

The prompt includes the schema: language tables for visible content, plain-string
contacts, tags before subtables, valid TOML escaping, and empty arrays for missing
sections. It asks the LLM to preserve source claims, dates, titles, and metrics.
Review the output before saving it; these instructions cannot enforce factual accuracy.

Use the emitted vocabulary in your profile's `tags` so relevant entries survive filtering.
Render, then inspect the PDF and its extracted text using the [scanner guide](scanners.md).

## Output and scripting

`run` writes to stdout. `--output <file>` writes a private file; `--copy` copies via
`wl-copy` or `xclip`. These two options are mutually exclusive.

`--json` gives stable JSON objects for `list`, `show`, and `run`. `--no-input` skips
interactive entry and bounds the initial stdin wait. Once piped data starts arriving,
stdin is read to EOF; a producer must close its stream for the command to finish.
Missing required inputs produce named errors.

Exit codes: `0` success, `1` resolution error (missing input, no clipboard tool),
`2` usage error.

```sh
resumegen prompt run analyze-jd --jd job.txt --json --no-input
resumegen prompt run cover-letter --app <id>
resumegen prompt run import-resume --resume my-resume.txt --output import-prompt.md --no-input
```
