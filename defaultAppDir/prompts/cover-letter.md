+++
name        = "cover-letter"
description = "Draft a targeted cover letter from your resume and a job description"

[inputs.resume]
source   = "data-dump"
required = true

[inputs.jd]
source   = "jd-file"
flag     = "jd"
required = true

[inputs.company]
source   = "flag"
flag     = "company"
required = true

[inputs.tone]
source   = "flag"
flag     = "tone"
required = false
default  = "warm and professional"
+++
Write a cover letter for a role at {{company}} in a {{tone}} tone.

My resume:

{{resume}}

The job description:

{{jd}}

Requirements:
- Three to four short paragraphs, under one page.
- Open with why this specific role, not a generic hook.
- Draw only on real experience from my resume; never invent achievements.
- Connect two or three of my strongest, most relevant accomplishments to what the JD asks for.
- Close with a clear, confident call to action.
