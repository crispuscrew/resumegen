# Resume data

[Back to README](../README.md)

Names, titles, companies, dates, locations, subtitles, degrees, skill names, summaries,
and bullet text use **language tables**, not bare strings. Any language key works;
`en` is the fallback. An omitted field renders empty.

`tags` is flat and must appear **before** the first subtable in an entry. Otherwise
TOML assigns it to the preceding subtable. Contacts are the exception to language
tables: their `lang`, `value`, and `href` fields are plain strings.

Bullet text, header summaries, and skill item names pass through the inline-markup
sanitizer. They support `*bold*`, `_italic_`, `` `code` ``, and `#link("url")[text]`.
Other fields render as plain strings; markup there shows up literally. Prefer plain
skill names so extracted keywords remain clear.

## jobs.toml

Job-level tags control the whole position; bullet-level tags control individual bullets.
A job without top-level tags is judged by its bullets. Keep entries newest first;
resumegen preserves authored order rather than sorting free-text dates.

```toml
[[jobs]]
tags = ["go", "backend", "devops"]

[jobs.company]
en = "Acme Corp"
ru = "Акме"

[jobs.title]
en = "Software Engineer"
ru = "Инженер-программист"

[jobs.date]
en = "Jan. 2025 - Present"

[jobs.location]
en = "Berlin, Germany"

[[jobs.bullets]]
tags = ["go", "backend"]
[jobs.bullets.text]
en = "Built a *REST API* service in Go"
ru = "Разработал сервис *REST API* на Go"
```

## header.toml

`[name]`, `[summary]`, and a list of contacts. A contact with `lang` shows only for that
language; without it, the contact always appears. `value` and `href` are plain strings,
so use separate language-specific entries when a contact needs translation.

```toml
[name]
en = "Ivan Petrov"

[[contacts]]
value = "you@example.com"
href = "mailto:you@example.com"

[[contacts]]
value = "linkedin.com/in/yourhandle"
href = "https://linkedin.com/in/yourhandle"

[[contacts]]
lang = "ru"
value = "t.me/yourhandle"
href = "https://t.me/yourhandle"

[summary]
en = "Go engineer with a background in systems programming"
```

An empty `href` renders the value as plain text. Non-empty links must use an allowed
URL scheme. Keep the actual email address, phone number, or profile URL visible in
`value` rather than using labels such as "Email" or "LinkedIn".

## projects.toml

Use `[[projects]]` with `tags`, `title`, `date`, `subtitle`, `detail`, and `bullets`.
The schema resembles jobs, but projects have no `company` or `location` fields.
`subtitle` is usually a tech stack; `detail` is usually a repository URL.
In the default template, subtitle, date, and detail share a metadata line below the title.
Omit `date` when the source has no dates for a personal project.

## skills.toml

Item names use language tables too, even though technology names rarely differ per
language. A lone `en` key is the norm. Empty visible categories are dropped.

```toml
[[categories]]
[categories.name]
en = "Languages"
ru = "Языки программирования"

[[categories.items]]
tags = ["go", "backend"]
[categories.items.name]
en = "Go"
```

## education.toml

Entries under `[[edu]]` use `title` (institution), `degree`, `location`, and `date`
language tables. Education is always shown in full, without tag filtering.

## Empty sections

All five data files must exist, even when a section has no entries. Use `jobs = []`,
`projects = []`, `edu = []`, or `categories = []` for an empty file. An absent summary
or empty education list does not produce an empty section heading in the default PDF.
