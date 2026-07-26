+++
name        = "rejection-analysis"
description = "Turn a rejection into concrete, honest lessons"

[inputs.resume]
source   = "data-dump"
required = false

[inputs.jd]
source   = "jd-file"
flag     = "jd"
required = false

[inputs.context]
source   = "prompt"
required = true
+++
I was rejected from a role. Help me learn from it without spiraling.

What happened, in my words:

{{context}}

Additional context, if present below (a job description and/or my background,
either of which may be empty; work with whatever is provided):

{{jd}}

{{resume}}

Give me a grounded analysis:

1. The **most likely reasons** for the rejection given what I described. Separate what I can control from what I cannot.
2. What, if anything, my resume or positioning could have done better for this specific role.
3. **Two or three concrete actions** for the next application, ranked by leverage.
4. A brief reality check: if this was mostly fit or luck rather than a fixable weakness, say so.

Be honest but constructive. Do not invent reasons the evidence does not support.
