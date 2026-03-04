// ─── Page & base typography ───────────────────────────────────────────────────

#let resume-init(body) = {
  set page(
    paper:  "us-letter",
    margin: (top: 0.5in, bottom: 0.5in, x: 0.5in),
  )
  set text(font: "New Computer Modern", size: 11pt)
  set par(leading: 0.55em, spacing: 0.55em)
  body
}

// ─── Section heading ──────────────────────────────────────────────────────────

#let section(title) = block(
  above: 12pt,
  below: 5pt,
  width: 100%,
  stack(
    dir: ttb,
    spacing: 3pt,
    text(size: 12pt, weight: "bold", smallcaps(title)),
    line(length: 100%, stroke: 0.5pt + black),
  ),
)

// ─── Two-row entry ────────────────────────────────────────────────────────────
// a / b  →  bold title        right-aligned date
// c / d  →  italic subtitle   right-aligned detail
// items  →  optional bullet list of (text: content) dicts

#let entry(a, b, c, d, items: none) = {
  v(9pt, weak: true)
  grid(
    columns:    (1fr, auto),
    row-gutter: 4pt,
    strong(a),                        align(right, b),
    emph(text(size: 9.5pt, c)),       align(right, emph(text(size: 9.5pt, d))),
  )
  if items != none and items.len() > 0 {
    v(5pt)
    pad(
      left: 12pt,
      list(
        marker:  sym.bullet,
        spacing: 5pt,
        ..items.map(i => text(size: 9.5pt, i.text))
      ),
    )
  }
}

// ─── Skill row ────────────────────────────────────────────────────────────────

#let skill(cat, val) = {
  text(size: 9.5pt)[*#cat*: #val]
  linebreak()
}
