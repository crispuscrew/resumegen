+++
name        = "tailor-bullets"
description = "Rewrite resume bullets to foreground what a job description values"

[inputs.resume]
source   = "data-dump"
required = true

[inputs.jd]
source   = "jd-file"
flag     = "jd"
required = true
+++
You are an expert resume editor. Here is my current resume:

{{resume}}

Here is the target job description:

{{jd}}

Rewrite my experience bullets so the most relevant work comes first and uses the
language of the job description. For each bullet you change, keep it to one line,
lead with a strong verb, and quantify impact where the original already gives you
the numbers.

Hard rules:
- Do not invent employers, titles, dates, technologies, or metrics.
- Only reuse facts present in my resume; if something is missing, say so instead of fabricating.
- Return the rewritten bullets grouped under their original job, plus a short note on what you changed and why.
