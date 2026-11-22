# Study Protocol v0.1 (DRAFT, not yet preregistered)

Title: Reach drift in automation and agent identities, a public repository measurement study
Author: Mayank Sekhar. Reach drift concept proposed by rpmsft9, definition pending agreement.
Status: draft. Definitions are provisional until aligned with OWASP NHI Top 10 issues 35 and 36.

## 1. Research questions
* RQ1 Prevalence: what share of automation identities show at least one reach drift event in a 12 month window?
* RQ2 Rate: what is the Reach Drift Rate, per cohort?
* RQ3 Detection gap: what share of drift events occur on identities that a static posture score rates low at the time of the event?
* RQ4 Coverage: which NHI 2025 and Agentic 2026 categories are observable from repository evidence, and which are not?

## 2. Definitions (draft)
* Identity unit: a workflow file (Cohort A) or an agent config file (Cohort B) at one path, tracked across its history.
* Tier: declared permissions block, OIDC audience and role, credential type, and the file owner.
* Reach set: external capabilities the identity can invoke. This means third party actions and reusable workflows, secrets and environments referenced, and declared MCP servers.
* Drift event: a commit where the reach set gains at least one element and the tier is unchanged.
* Normalization: a version bump of an already referenced action is not growth. Bot authored pin updates are excluded from counts and reported separately.
* Reach Drift Rate: drift events per identity per 90 days of observed history.

## 3. Cohorts and sampling
* Cohort A: public repos with at least one workflow file.
* Cohort B: public repos with an MCP or agent config file. Exploratory, because these files are recent and rare.
* Inclusion: not a fork, not archived, at least 20 commits, at least 12 months of history, at least 2 commits touching the identity file.
* Sampling: stratified random draw by activity band and star band. Record the date of the sampling frame.
* Size: pilot of 50 repos to estimate prevalence. Then size to a 95% CI of plus or minus 5 points, about 385 repos at worst case prevalence. Cohort B is reported descriptively.

## 4. Extraction
Parse every commit that touches the identity file, build reach and tier per commit, and diff against the previous version. Store only derived sets, counts, commit hash and timestamp. Never store file bodies or secret values.

## 5. Baseline
Adapt the engine's static factors to repository evidence (tier, credential type, owner present) and compute the score at each commit. Freeze and publish the adaptation before data collection. Report the share of drift events that occur while the static score is below the median.

## 6. Validation
Two raters label 200 flagged events as true drift, false positive (with reason), or ambiguous, using a written codebook. Report precision with a 95% CI and Cohen's kappa. Also label 200 unflagged config commits to estimate misses.

## 7. Ethics and data handling
Public data only. No secret values retained. Aggregate reporting, with no repos named in negative findings. Respect rate limits. If a live secret is seen during review, do not store it and notify the owner. Check GitHub terms for research use before collection.

## 8. Threats to validity
* Config is a proxy for runtime reach.
* MCP tool lists are often resolved at runtime, so only declared servers are observable.
* Public repos skew small.
* Bots and renames can inflate counts.
* Workflows are automation identities, not all agents. Cohort A supports the NHI claim. Cohort B supports the agent claim only weakly.

## 9. Preregistration and open decisions
Freeze by tagging this file and registering it (for example on OSF) before any data collection.
Open:
* Definition alignment with rpmsft9 (issues 35 and 36)
* Tier composition
* Whether to include GitLab
* Minimum Cohort B size
* Identity of the second rater
* Whether Cohort A is the main sample (recommended)
