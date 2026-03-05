# resumegen — TODO

Infrastructure (configs, templates, data) is done. Everything below is Go code to write.

## 1. Project scaffold
- [x] `go.mod` — module `github.com/crispuscrew/resumegen`, dep `github.com/BurntSushi/toml v1.4.0`
- [x] `embed.go` — `//go:embed defaults` → `var Defaults embed.FS`
- [ ] `go.sum` — `go mod tidy` after writing go.mod

## 2. `internal/config` — global config
- [x] Struct matching `defaults/config.toml`: `Paths`, `Render`, `Estimation` sections
- [ ] Loader: search `./config.toml` → `$XDG_CONFIG_HOME/resumegen/config.toml`
  - Not found → prompt user → unpack `Defaults` → exit (user edits and reruns)
  - Parse error → `fmt.Fprintf(os.Stderr, ...)` + `os.Exit(1)`

## 3. `internal/data` — resume content types + loader
- [ ] Types: `Header`, `Contact`, `Job`, `Bullet`, `SkillCategory`, `SkillItem`, `School`, `Project`
  - Bilingual fields: `type I18n struct { En, Ru string }`
  - Contact has optional `Lang string`
- [ ] Loader: read all five TOML files from `config.Paths.DataDir`

## 4. `internal/profile` — profile types + loader
- [ ] Struct: `Tags []string`, `Lang string`, `Output string`, `Trim []TrimGroup`
  - `TrimGroup`: `Tags []string`
- [ ] Loader: read `<profiles_dir>/<name>.toml`; fail hard on parse error

## 5. `internal/filter` — tag filtering
- [ ] `FilterBullets(bullets []Bullet, active []string) []Bullet`
- [ ] `FilterJobs(jobs []Job, active []string) []Job` — skips job if 0 visible bullets, respects job-level tags
- [ ] `FilterProjects(...)` — same logic as jobs
- [ ] `FilterSkills(cats []SkillCategory, active []string) []SkillCategory`
- [ ] `FilterContacts(contacts []Contact, lang string) []Contact`
- [ ] `ResolveI18n(field I18n, lang string) string` — picks en/ru string

## 6. `internal/trim` — page overflow trimming
- [ ] `Estimate(data ResumeData, cfg EstimationConfig) float64` — sum pt heights
  - Section header, entry header, per-bullet, summary lines, skill lines
  - Wrap estimation: `len(text) / charsPerLine` (rough, calibrate empirically)
- [ ] `TrimOnce(data *ResumeData, trimGroups []TrimGroup) bool`
  - Find lowest-priority group that still has bullets → drop last bullet of first matching job/project
  - Return false if nothing left to drop
- [ ] `FitsPage(estimate float64, cfg EstimationConfig) bool`

## 7. `internal/render` — data_gen.typ generator + typst runner
- [ ] `WriteDataGen(path string, data ResumeData)` — write `templates/data_gen.typ`
  - Escape `]` → `\]` in content strings before wrapping in `[...]`
  - Skills: join item names with `", "`
- [ ] `Compile(cfg Config, profile Profile) error` — exec `typst compile templates/resume.typ -o <output>`
- [ ] `PageCount(pdfPath string) (int, error)` — use `pdfcpu` or parse PDF binary

## 8. `cmd/resumegen/main.go` — wire everything
- [ ] `main()`: load config → load data → load profile → filter → estimate loop:
  ```
  for {
      if FitsPage(Estimate(data)) { break }
      if !TrimOnce(&data) { break }   // nothing left to drop
  }
  ```
- [ ] Generate `data_gen.typ` → compile → check page count (phase A):
  ```
  for {
      WriteDataGen(...)
      Compile(...)
      if PageCount(...) <= pageLimit { break }
      if !TrimOnce(&data) { break }
  }
  ```
- [ ] Clean up `data_gen.typ` after successful compile (or keep, up to you)
- [ ] `resumegen <profile>` usage, e.g. `resumegen go-backend`
- [ ] `resumegen init` — unpack defaults to CWD (no prompt, explicit command)

## 9. Containerfile / Makefile (I'll write these)
- [ ] Containerfile: Typst + Go binary, multi-stage build
- [ ] Makefile: `make go-backend`, `make go-backend LANG=ru`, `make all`, `make clean`

## Notes
- `data_gen.typ` goes into `templates/` dir (add to `.gitignore`)
- Page estimation constants are in `config.toml [estimation]` — tweak after first real compile
- `pdfcpu` for page count: `github.com/pdfcpu/pdfcpu/pkg/api` → `api.PageCount(r, nil)`
