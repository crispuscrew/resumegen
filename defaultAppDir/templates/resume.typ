// Data is supplied via data_gen.typ (auto-generated, do not edit).
#import "template.typ": *
#import "data_gen.typ": r-lang, r-name, r-contacts, r-summary, r-jobs, r-projects, r-skills, r-edu

#show: resume-init
#set text(lang: r-lang)
#set document(title: r-name + " - Resume")

#let translate(english, russian) = if r-lang == "ru" { russian } else { english }

// Contact details are visible body text, not a page header or icon labels.
#title(r-name)
#if r-contacts.len() > 0 {
  block(width: 100%, above: 0pt, below: 2pt, align(center, text(size: 10pt)[
    #for (index, contact) in r-contacts.enumerate() {
      if index > 0 [#h(4pt)|#h(4pt)]
      if contact.href == "" { contact.value } else { link(contact.href)[#contact.value] }
    }
  ]))
}

#if r-summary != [] {
  section(translate("Summary", "О себе"))
  block(above: 0pt, below: 0pt, text(size: 10pt, r-summary))
}

#if r-jobs.len() > 0 {
  section(translate("Work Experience", "Опыт работы"))
  for job in r-jobs {
    entry(job.title, job.date, job.company, job.location,
      items: job.bullets.map(bullet => (text: bullet,)))
  }
}

#if r-projects.len() > 0 {
  section(translate("Projects", "Проекты"))
  for project in r-projects {
    entry(project.title, project.date, project.subtitle, project.detail,
      items: project.bullets.map(bullet => (text: bullet,)))
  }
}

#if r-skills.len() > 0 {
  section(translate("Technical Skills", "Технические навыки"))
  block(above: 0pt, below: 0pt,
    for skills in r-skills {
      skill(skills.category, skills.items)
    }
  )
}

#if r-edu.len() > 0 {
  section(translate("Education", "Образование"))
  for education in r-edu {
    entry(education.title, education.date, education.degree, education.location)
  }
}

#context [#metadata(here().position()) <end-marker>]
