+++
name        = "import-resume"
description = "Turn a resume you already have into filled-in resumegen data files"

[inputs.resume]
source   = "file"
flag     = "resume"
required = true

[inputs.tags]
source   = "flag"
flag     = "tags"
required = false
default  = "(none supplied: derive a vocabulary from the resume itself and report it after the files)"

[inputs.langs]
source   = "flag"
flag     = "langs"
required = false
default  = "en"
+++
You are converting an existing resume into the TOML data files used by resumegen, a tool that
renders many targeted PDF resumes from one tagged dataset. Your output is consumed by a parser,
so the schema rules below are strict.

Source resume:
{{resume}}

Tag vocabulary to use: {{tags}}

Languages to emit: {{langs}}

## What to produce
Exactly five files, each in its own fenced code block labelled with its filename:
`header.toml`, `jobs.toml`, `projects.toml`, `education.toml`, `skills.toml`.

## Schema rules (all mandatory)

1. **Every human-visible field except contact values is a language table.** Write
   `[jobs.company]` with an `en = "..."` key under it, not `company = "Acme"`. This applies to
   names, titles, companies, dates, locations, subtitles, degrees, skill names, and bullet text.
   A bare string makes the file fail to load.
2. **`tags` must come before the first subtable of its entry.** Contact `lang`, `value`, and
   `href` are flat strings, not language tables.
   TOML assigns a flat key to whatever table precedes it, so `tags` written after `[jobs.company]`
   silently lands on the wrong entry.
3. **Emit one key per requested language** in every language table, using the codes listed above.
   `en` is the fallback whenever a requested language is missing. If only `en` is requested, emit
   only `en`. Never machine-translate a proper noun that is normally left in English (product
   names, technologies); do translate role titles, locations, and prose.
4. **Use inline markup in bullet text and the header summary only.** There you may use
   `*bold*`, `_italic_`, `` `code` ``, and `#link("url")[text]`. Keep all other fields plain,
   including skill names for clear keyword matching.
5. **Omit fields with no source value.** Do not invent placeholders. If a file has no entries,
   still emit it with an empty array (`jobs = []`, `projects = []`, `edu = []`, or `categories = []`).
   Escape quotes, backslashes, and newlines using valid TOML string syntax.
6. **Tags drive everything.** Each job, project, skill item, and individual bullet carries tags;
   a profile later selects content by them. Use a small, consistent, lower-case vocabulary
   (`go`, `backend`, `devops`, `cpp`, `robotics`, ...). Reuse the same tag for the same concept
   everywhere. Tag each bullet by what it actually demonstrates, not by where it happens to sit,
   because a bullet with no matching tag is dropped even when its job is kept.

## Shapes

```toml
# header.toml
[name]
en = "Ivan Petrov"
[[contacts]]
value = "you@example.com"
href  = "mailto:you@example.com"
[summary]
en = "Go engineer with a background in systems and C++ robotics"
```

```toml
# jobs.toml   (tags first, then subtables)
[[jobs]]
tags = ["go", "backend", "devops"]
[jobs.company]
en = "Acme Corp"
[jobs.title]
en = "Software Engineer"
[jobs.date]
en = "Jan. 2025 - Present"
[jobs.location]
en = "Berlin, Germany"
[[jobs.bullets]]
tags = ["go", "backend"]
[jobs.bullets.text]
en = "Built a *REST API* service in Go serving *12k RPC/s*"
```

```toml
# projects.toml   (same shape as jobs; title replaces company)
[[projects]]
tags = ["go", "backend"]
[projects.title]
en = "nightglass"
[projects.subtitle]                # tech-stack label on the metadata line
en = "Go, GStreamer, ARM Linux"
[projects.detail]                  # metadata text, usually a visible repo URL
en = "github.com/you/nightglass"
[[projects.bullets]]
tags = ["go"]
[projects.bullets.text]
en = "On-device detection under a *`<100 ms`* latency budget"
```

```toml
# education.toml   (no tags anywhere; always shown in full)
[[edu]]
[edu.title]
en = "Example State University"
[edu.degree]
en = "M.Sc., Computer Science"
[edu.location]
en = "Berlin, Germany"
[edu.date]
en = "2020 - 2024"
```

```toml
# skills.toml
[[categories]]
[categories.name]
en = "Languages"
[[categories.items]]
tags = ["go", "backend"]
[categories.items.name]
en = "Go"
```

## Content guidance

- Keep every claim traceable to the source resume. Do not invent employers, dates, metrics, or
  technologies, and do not upgrade a title. If the source is ambiguous, keep the wording vague
  rather than guessing a specific.
- Split run-on responsibilities into separate bullets so each can be tagged and kept or dropped
  independently. One bullet should carry one accomplishment.
- Lead bullets with the action and keep any number the source gives, bolding the figure that
  matters: `Cut p99 latency to *40 ms*`.
- Order jobs newest first.
- Keep email addresses, phone numbers, and profile URLs visible as contact values, not
  generic labels like "LinkedIn". Preserve month/year dates and use "Present" for ongoing roles
  only when the source says they are ongoing. Do not add keywords unsupported by the source.

After the five blocks, list the tag vocabulary you used, one line per tag with a short gloss, plus
a note of anything in the source resume you could not place into the schema.
