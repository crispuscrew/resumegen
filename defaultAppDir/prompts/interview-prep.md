+++
name        = "interview-prep"
description = "Generate likely interview questions and answer scaffolds"

[inputs.resume]
source   = "data-dump"
required = true

[inputs.jd]
source   = "jd-file"
flag     = "jd"
required = true
+++
You are preparing me for an interview for the role below. Use my resume and the
job description to anticipate what I will be asked.

My resume:

{{resume}}

The job description:

{{jd}}

Produce:

1. **Technical questions** (6-8) the JD implies, hardest first.
2. **Behavioral questions** (4-6), each tied to a real accomplishment from my resume I can use as a STAR answer.
3. **Questions I should ask them**, specific to this role and team.
4. **My likely weak spots** given the gaps between resume and JD, and how to address them without bluffing.

Anchor every suggested answer in real experience from my resume. Flag anything I would need to be able to back up.
