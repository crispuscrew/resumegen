+++
name        = "followup"
description = "Write a follow-up note after applying or interviewing"

[inputs.company]
source   = "flag"
flag     = "company"
required = false
default  = "the company"

[inputs.role]
source   = "flag"
flag     = "role"
required = false
default  = "the role"

[inputs.context]
source   = "prompt"
required = true
+++
Write a follow-up message for my application to {{role}} at {{company}}.

Here is where things stand, in my own words:

{{context}}

Requirements:
- Short: a recruiter should read it in fifteen seconds.
- Reaffirm genuine interest and add one specific, relevant detail (not a generic "just checking in").
- Match the stage I described: post-application, post-interview, or chasing a silent thread.
- Polite and low-pressure; never entitled. Do not invent conversations or commitments that did not happen.
