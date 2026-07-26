+++
name        = "salary-research"
description = "Prepare a compensation range and negotiation stance for a role"

[inputs.role]
source   = "flag"
flag     = "role"
required = true

[inputs.location]
source   = "flag"
flag     = "location"
required = false
default  = "a remote / unspecified location"

[inputs.resume]
source   = "data-dump"
required = false
+++
Help me prepare compensation expectations for a "{{role}}" role in {{location}}.

For context on my level, here is my background:

{{resume}}

Provide:

1. A realistic **base salary range** (low / median / high) for this role and location, and the assumptions behind it.
2. How **total compensation** typically breaks down (base, bonus, equity) at this level.
3. The factors that would push me toward the top or bottom of the range, based on my background.
4. A concrete **negotiation stance**: my anchor number, my walk-away, and two non-salary levers worth trading for.

Be explicit that these are estimates and that I should corroborate with current, source-based data (levels.fyi, Glassdoor, peers). Do not present guesses as facts.
