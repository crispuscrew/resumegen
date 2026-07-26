+++
name        = "recruiter-reply"
description = "Draft a reply to a recruiter's outreach message"

[inputs.message]
source   = "stdin"
required = true

[inputs.resume]
source   = "data-dump"
required = false

[inputs.intent]
source   = "flag"
flag     = "intent"
required = false
default  = "interested and want to learn more"
+++
A recruiter sent me this message:

{{message}}

If a background summary is provided below, use it for context (it may be empty):

{{resume}}

Draft a reply. My intent: {{intent}}.

Keep it short, courteous, and specific. If I am interested, ask the two or three
questions that most determine fit (scope, compensation range, work model). If I am
declining, be gracious and leave the door open. Do not overstate my experience
beyond what my background supports.
