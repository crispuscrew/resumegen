+++
name        = "gap-report"
description = "Find the gaps between your resume and a target job description"

[inputs.resume]
source   = "data-dump"
required = true

[inputs.jd]
source   = "jd-file"
flag     = "jd"
required = true
+++
Compare my resume against this job description and report the gaps honestly.

My resume:

{{resume}}

The job description:

{{jd}}

Produce three lists:

1. **Strong matches**: requirements my resume already demonstrates, with the evidence.
2. **Partial matches**: requirements I touch but under-sell; note how I could surface them better using only real experience.
3. **Genuine gaps**: requirements I show no evidence for. For each, say whether it is likely a dealbreaker and what the fastest credible way to close it would be.

Do not paper over a gap by inventing experience. If I am not a fit, say so plainly.
