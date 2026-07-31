// resume.typ - main document.
//
// Data is supplied by the Go tool via data_gen.typ (auto-generated, do not edit).
// Run `resumegen <profile>` to regenerate and compile.

#import "template.typ": *
#import "data_gen.typ": r-lang, r-name, r-contacts, r-summary, r-jobs, r-projects, r-skills, r-edu

#show: resume-init

// ─── i18n ─────────────────────────────────────────────────────────────────────

#let t(en, ru) = if r-lang == "ru" { ru } else { en }

// ============================================================================
//  Header
// ============================================================================

#align(center)[
  #text(size: 22pt, weight: "bold", smallcaps(r-name)) \
  #v(2pt)
  #text(size: 10pt)[
    #for (idx, c) in r-contacts.enumerate() {
      if idx > 0 [ #h(4pt) | #h(4pt) ]
      if c.href == "" { c.value } else { link(c.href)[#c.value] }
    }
  ]
]

// ============================================================================
//  Summary
// ============================================================================

#section(t("Summary", "О себе"))
#block(above: 0pt, below: 0pt, text(size: 10pt, r-summary))

// ============================================================================
//  Experience
// ============================================================================

#if r-jobs.len() > 0 {
  section(t("Experience", "Опыт работы"))
  for job in r-jobs {
    entry(job.title, job.date, job.company, job.location,
      items: job.bullets.map(b => (text: b,)))
  }
}

// ============================================================================
//  Projects
// ============================================================================

#if r-projects.len() > 0 {
  section(t("Projects", "Проекты"))
  for p in r-projects {
    entry(p.title, p.date, p.subtitle, p.detail,
      items: p.bullets.map(b => (text: b,)))
  }
}

// ============================================================================
//  Technical Skills
// ============================================================================

#if r-skills.len() > 0 {
  section(t("Technical Skills", "Технические навыки"))
  block(above: 0pt, below: 0pt,
    for s in r-skills {
      skill(s.category, s.items)
    }
  )
}

// ============================================================================
//  Education
// ============================================================================

#section(t("Education", "Образование"))
#for edu in r-edu {
  entry(edu.title, edu.location, edu.degree, edu.date)
}

#context [#metadata(here().position()) <end-marker>]