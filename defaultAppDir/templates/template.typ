// Page and base typography.
#let resume-init(body) = {
  set page(
    paper: "a4",
    margin: (top: 0.5in, bottom: 0.5in, x: 0.5in),
  )
  // These fonts ship with Typst and cover Latin and Cyrillic.
  set text(
    font: ("New Computer Modern", "Libertinus Serif"),
    size: 11pt,
    ligatures: false,
    hyphenate: false,
  )
  show raw: set text(font: "DejaVu Sans Mono")
  set par(leading: 0.55em, spacing: 0.55em)
  show title: set text(size: 22pt, weight: "bold")
  show title: set align(center)
  show title: set block(above: 0pt, below: 6pt)
  show heading: set text(size: 12pt, weight: "bold")
  show heading: heading => block(
    width: 100%, above: 12pt, below: 6pt, breakable: false, sticky: true,
    stack(dir: ttb, spacing: 3pt,
      upper(heading.body),
      line(length: 100%, stroke: 0.35pt + black),
    ),
  )
  body
}

// Real headings preserve section semantics in the tagged PDF.
#let section(title) = heading(level: 1, numbering: none, title)

// Keep the positional interface for custom resume.typ files. All fields flow
// left to right, top to bottom. Keep metadata together so extraction cannot
// mistake distant right-aligned dates for a separate text column.
#let entry(title, date, subtitle, detail, items: none) = {
  let metadata = (
    (value: subtitle, emphasis: true),
    (value: date, emphasis: false),
    (value: detail, emphasis: true),
  ).filter(field => field.value != "")
  block(above: 9pt, below: 0pt, breakable: false)[
    #strong(title)
    #if metadata.len() > 0 {
      linebreak()
      text(size: 9.5pt)[
        #for (index, field) in metadata.enumerate() {
          if index > 0 [#h(3pt)|#h(3pt)]
          if field.emphasis { emph(field.value) } else { field.value }
        }
      ]
    }
  ]
  if items != none and items.len() > 0 {
    v(5pt)
    list(
      marker: sym.bullet,
      indent: 8pt,
      body-indent: 10pt,
      spacing: 5pt,
      ..items.map(item => text(size: 10pt, item.text))
    )
  }
}

#let skill(category, items) = {
  text(size: 10pt)[*#category*: #items.join(", ")]
  linebreak()
}
