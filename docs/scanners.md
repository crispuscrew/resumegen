# Scanner compatibility

[Back to README](../README.md)

The bundled layout is designed for text-based resume imports and applicant tracking
systems (ATS), including applications submitted through LinkedIn. A local extraction
check confirms readable text; it does not guarantee a particular platform's parsing,
ranking, or matching behavior. Review the populated fields after each upload.

## Layout choices

- One text column with a consistent top-to-bottom reading order, without grids,
  tables, sidebars, icons, or page-header contact details. Dates stay next to the
  location so extraction cannot mistake them for a separate column.
- Real Typst section headings: Summary, Work Experience, Projects, Technical Skills,
  and Education. Empty summary and education sections are omitted.
- Each role, project, or institution has its own bold title line. A smaller metadata
  line groups its employer/stack/degree, dates, and location/repository underneath.
  Dates and separators stay upright; supporting labels use italics.
- A centered name/contact block, uppercase section headings, thin rules, and indented
  bullets establish visual hierarchy. The template uses New Computer Modern, included
  with Typst: 11pt entry titles, 10pt body text, and 9.5pt metadata.
- Actual email addresses and profile URLs visible in the text, with clickable links.
- Embedded fonts, no small capitals, no automatic hyphenation, and no typographic
  ligatures that could complicate extracted keyword text.
- The PDF declares the profile language and a document title, and uses native
  headings/lists so Typst can preserve semantic tags.

These choices follow Typst's guidance on [semantics and reading order](https://typst.app/docs/guides/accessibility/).

## Readability and parsing: research

Sources checked on October 1, 2026:

- [TRUE] [Harvard's resume guide](https://careerservices.fas.harvard.edu/resources/create-a-strong-resume/)
  recommends balancing whitespace and using consistent bold, italics, and spacing.
- [TRUE] [Greenhouse's parser guidance](https://support.greenhouse.io/hc/en-us/articles/200989175-Unsuccessful-resume-parse)
  identifies columns, complex tables, graphics, and header/footer contact details as
  potential causes of incomplete parsing. It also calls for clear, consistent sections.
- [TRUE] [Jobscan's formatting guide](https://www.jobscan.co/blog/20-ats-friendly-resume-templates/)
  recommends 10-12pt body text, standard bullets, and a text-based PDF or DOCX.
- [MIXED] These sources support readable, restrained formatting; they do not validate
  this template against every ATS. A local right-aligned-date trial reordered dates
  in ordinary PDF extraction, so dates remain inline. Tests check both visual and
  source-order text, but platform upload checks remain necessary.

Reliability: High for the cited guidance; platform compatibility remains unverified.

## Content

Use consistent month/year employment dates and newest-first entries. Write the real
job title and employer name. Include relevant technologies and accomplishments that
are supported by your experience. Profile tags select content but are not printed;
keywords need to appear naturally in the visible resume text.

## Check your PDF

```sh
pdftotext output/default.pdf -
pdftotext -layout output/default.pdf -
pdfinfo output/default.pdf
```

Check that your name, contacts, job titles, employers, dates, skills, and education
appear in order and that important words stay intact. Copy/paste into a plain-text
editor is another quick check. Upload the PDF, not the PNG preview.

## Existing workspaces

Local template copies shadow the bundled ones. Back up any customized
`templates/resume.typ` and `templates/template.typ`, then extract the new defaults:

```sh
resumegen template extract resume.typ template.typ --path /path/to/your/workspace
resumegen --path /path/to/your/workspace --profile default
```

Extraction never overwrites existing copies. To adopt the new layout, move your saved
copies aside before extracting, or merge the layout changes into your custom files.
The positional `entry(title, date, subtitle, detail, items: ...)` interface is retained.
