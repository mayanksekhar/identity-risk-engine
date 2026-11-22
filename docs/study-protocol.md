# Study Protocol v0.3 (DRAFT, not yet preregistered)

Title: Reach drift in automation and agent identities, a public repository measurement study
Author: Mayank Sekhar. Reach drift concept proposed by rpmsft9, definition pending agreement.
Status: draft v0.3. Definitions are provisional until aligned with OWASP NHI Top 10 issues 35, 36 and 43.

## 1. Research questions
* RQ1 Prevalence: what share of workflow histories, and of repositories, show at least one reach drift event in the observation window?
* RQ2 Rate: what is the Reach Drift Rate?
* RQ3 Detection gap: (a) what share of drift events occur on identities that a static posture score rates below the median at the time of the event, and (b) what share of reach growth coincides with a tier increase (visible to static scoring) versus no tier change (invisible)?
* RQ4 Coverage: which NHI 2025 and Agentic 2026 categories are observable from workflow evidence, and which are not?

## 2. Definitions (draft)
* Identity unit: a workflow history, meaning one workflow file tracked across its versions and renames using the dataset's `uid` column. A file deleted and re added receives a new uid.
* Reach set at a version: distinct external capabilities the workflow references. These are third party actions and reusable workflows (identified by owner/repo[/path], ignoring the ref), secret names referenced, and environments referenced.
* Tier at a version: the declared permissions at workflow and job level, plus the `id-token` permission. A missing permissions key is recorded as "default". The real default depends on repository or organization settings and is not observable. File owner is not observable in the dataset and is excluded.
* Tier change direction: up (any permission widened or `id-token` write added), down (narrowed), or none.
* Gain: an element in the reach set that was absent in the previous version. A version only change (same owner/repo path, different ref) is not a gain. A replacement counts as one gain plus one loss, reported separately.
* Drift event: a commit by a non bot author with at least one gain and tier change direction none.
* Visible growth event: a commit with at least one gain and tier change direction up.
* Reach Drift Rate: drift events per workflow history per 90 days observed.
* Pin type per element: SHA, full tag, major tag, branch, or other. Recorded at every version.
* Bot commit: `author_name` or `author_email` matches a published bot pattern list (for example dependabot, renovate, names ending in [bot]). Bot commits are excluded from counts and reported separately.

## 3. Cohorts and data
* Cohort A (main sample): the University of Mons dataset 'A dataset of GitHub Actions workflow histories' (Cardoen, Mens, Decan, MSR 2024), version 2026-05-22, Zenodo record 20340547, licence CC BY 4.0. Version DOI 10.5281/zenodo.20340547 (inferred from the record number, to be confirmed). Concept DOI 10.5281/zenodo.10259013. Repositories: non fork, created at least one year before the pull, at least 300 commits, at least 100 stars, at least one commit in the year before the pull (26 March 2026). Files used: repositories.csv.gz, workflows.csv.gz, workflows.tar.gz. The Mons evolution paper (arXiv 2602.14572) used the 2025-10-09 version and reports a 300 star threshold. We follow the record for the pinned version.
* Inclusion: valid_yaml is true on every version of the history, and the history exists at some point in the window. Histories excluded for invalid YAML are counted and reported.
* Observation window: 26 March 2025 to 26 March 2026, the 12 months before the pull date.
* Pilot: a random sample of 1,000 histories to debug extraction and the codebook. Pilot histories are held out of confirmatory estimates.
* Analysis: the full eligible population. Confidence intervals use a bootstrap clustered by repository, because histories within a repository are not independent.
* Cohort B (exploratory, later): agent and MCP configuration files from a small crawl of our own. A protocol addendum will be written before any collection. Descriptive reporting only.
* Out of scope: GitLab and other platforms.

## 4. Extraction
Reuse the dataset's commit metadata and file contents. Reach and tier are computed by our own parser applied to each version, with the dataset's change sets used for cross checks. Store derived data only: sets, counts, pin types, commit hash, timestamp. Never store secret values. Secret names only.

## 5. Baseline
Adapt the engine's static factors to workflow evidence (permission breadth, credential type, trigger exposure) and compute the score at each version. Freeze and publish the adaptation before the full run. Report the share of drift events that occur while the static score is below the median.

## 6. Validation
Two raters label 200 flagged events as true drift, false positive (with reason), or ambiguous, using a written codebook. Report precision with a 95% CI and Cohen's kappa. Also label 200 unflagged commits that touch reach or tier, to estimate misses.

## 7. Ethics and data handling
Public data only. The dataset is CC BY 4.0, so cite Cardoen, Mens and Decan (MSR 2024) and the dataset version, and state any changes. CC BY covers the dataset compilation, not the licences of the underlying repositories, so we release derived tables, not file bodies. The dataset contains author names and emails, which we do not release. Derived tables carry a bot flag only. Aggregate reporting, with no repositories named in negative findings. If a live secret is seen during review, do not store it and notify the owner.

## 8. Threats to validity
* Config is a proxy for runtime reach.
* The dataset covers popular, active repositories only (at least 100 stars and 300 commits), so results may not generalize to small projects.
* Primary branch only, and squash merges can hide separate changes.
* Histories with invalid YAML are excluded, as in the source study.
* A workflow deleted and re added gets a new uid, so a replacement looks like a new identity and its reach is not compared with the old one.
* Reach can change with no commit when a reference is a mutable tag or branch. We record pin type and report the share of reach under mutable references, but do not count such changes as drift.
* A missing permissions key hides the real default token permissions.
* Bot detection by author name is imperfect.
* Workflows are automation identities, not all agents. Cohort A supports the NHI claim. Cohort B supports the agent claim only weakly.

## 9. Related work (preliminary, not systematic)
* Rostami Mazrae et al., arXiv 2602.14572: workflow evolution at scale, permission additions, version pinning trends, bot updates. No reach versus tier framing.
* Vendor analysis of the top 2,000 repositories reporting overly permissive workflows: a static snapshot.
* Hush Security report on about 82,000 public MCP configs: credential prevalence, vendor report.
* Studies of Actions marketplace evolution and dependency rug pulls (SecDev 2025, ICSA 2026): read from abstracts only.
* Cardoen, Mens, Decan, MSR 2024: the dataset paper.
* Talebzadeh Bardsiri, Decan, Mens, arXiv 2605.26825: usage and evolution of workflow language constructs and reliability. Read from the abstract only.
* A systematic search with logged terms and databases is still to be done before preregistration.

## 10. Preregistration and open decisions
Sequence: pilot, freeze definitions, preregister (for example on OSF), then the full run.
Decided: reuse the Mons dataset for Cohort A. Cohort B comes later from our own crawl. Cohort A is the main sample.
Open:
* Confirm the version DOI on the record page
* Definition alignment with rpmsft9, including what "tier" means in nhi-scan
* Identity of the second rater
* Systematic literature search

## 11. Decision log
* 23 November 2026: reuse the Mons dataset for Cohort A, to save collection time and improve reproducibility. Accepted cost: popularity bias.
* 23 November 2026: exclude bot version bumps and define gain by owner/repo path, since bots account for most version updates in the source study.
* 23 November 2026: pin dataset version 2026-05-22 (Zenodo record 20340547, CC BY 4.0). Observation window 26 March 2025 to 26 March 2026. Star threshold corrected to 100, from the record, not the paper.

## 12. Frozen operationalization (v0.4)

Status: frozen on 23 November 2026, after the 1,000 history pilot and before
any output of the full run was read. The pilot histories (the first 1,000 of
the sorted uid list shuffled with seed 20261123) are excluded from every
confirmatory estimate. Any later change to this section is a dated amendment
with a reason, and the affected analysis is labelled as post hoc.

### 12.1 Window and exposure
The window runs from 2025-03-26 00:00:00 UTC to 2026-03-26 23:59:59 UTC.
committed_date is epoch seconds in UTC. A history enters exposure at the later
of the window start and its first commit. It leaves at the earlier of the
window end and its delete commit. Histories deleted before the window are
excluded and counted. Histories with no in window change stay in as zero event
exposure. The rate is drift events divided by exposure days over 90.

### 12.2 Ordering
Rows are sorted by committed_date. The CSV direction is detected per history
and reversed when it is newest first, then the sort is stable. Histories with
timestamp ties are counted, and a sensitivity analysis drops them.

### 12.3 State and baseline
The baseline is the last content version before the window start. A history
added inside the window uses its first content version as the baseline, which
is not an event. A rename row with an empty file_hash leaves state unchanged.
A delete row ends the history. Pin changes alone are not events.

### 12.4 Parsing
Files are parsed with the PyYAML safe loader. An unparsable version is skipped
and the last good state is carried forward. The count of unparsable versions
is reported per history and in total.

### 12.5 Reach set
The reach set is the set of owner/repo[/path] strings, lowercased with the ref
removed, taken from job level uses and step level uses. Local paths (./) and
docker:// images are excluded. The primary analysis counts third party reach,
meaning every owner except the official owners actions and github. The
sensitivity analysis counts all external owners.

### 12.6 Tier
Each permissions value maps to a level: none 0, read 1, default 2, write 3.
Missing permissions mean default. A job level permissions block overrides the
workflow level block. The tier is two times the highest level across jobs,
plus one when any scope sets id-token to write. This is our operationalization.
Alignment with the definition used by nhi-scan is pending, and results are
reported as ours.

### 12.7 Events
A drift event is a commit by a non bot author with a third party reach gain
greater than zero and an unchanged tier. A visible growth event is the same
with the tier raised. Other combinations are reported but are not primary.

### 12.8 Pin types
sha is 40 hex characters. full starts with digits, a dot and digits, with an
optional leading v. major is an optional v plus digits only. branch is
everything else, including a missing ref. When one action appears more than
once in a file, the weakest pin is kept.

### 12.9 Bots
A commit is a bot commit when the lowercased author name plus author email
contains any of: dependabot, renovate, [bot], github-actions, greenkeeper,
snyk-bot. Only the flag is released, never names or emails.

### 12.10 Estimands and inference
The primary estimand is the Reach Drift Rate per history per 90 days.
Prevalence is the share of exposed histories with at least one drift event.
Intervals are 95 percent percentile intervals from a cluster bootstrap by
repository with 10,000 resamples.

## 11a. Decision log additions
- 2026-11-23: pilot of 1,000 histories run to debug the extractor only. 791
  had exposure, 10 of 4,156 files were unparsable, none were missing.
- 2026-11-23: section 12 frozen before reading full run output.
- 2026-11-23: primary reach scope set to third party (non official owners) to
  match the wording of section 2. All external owners is a sensitivity analysis.
