+++
name        = "analyze-jd"
description = "Break a job description into requirements, keywords, and signals"

[inputs.jd]
source   = "jd-file"
flag     = "jd"
required = true
+++
You are an experienced technical recruiter. Analyze the following job description.

Job description:

{{jd}}

Produce:

1. **Must-have requirements**: hard requirements the candidate cannot lack.
2. **Nice-to-have requirements**: preferred but not disqualifying.
3. **Keywords & technologies**: the exact terms an ATS or reviewer will scan for.
4. **Seniority & scope**: what level this role really targets, with evidence from the text.
5. **Red / green flags**: anything in the wording that hints at team health, workload, or expectations.

Be concrete and quote the JD where it matters. Do not invent requirements that are not present.
