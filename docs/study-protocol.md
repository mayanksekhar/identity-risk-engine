# Study Protocol v0.2 (DRAFT, not yet preregistered)

Title: Reach drift in automation and agent identities, a public repository measurement study
Author: Mayank Sekhar. Reach drift concept proposed by rpmsft9, definition pending agreement.
Status: draft v0.2. Definitions are provisional until aligned with OWASP NHI Top 10 issues 35, 36 and 43.

## 1. Research questions
* RQ1 Prevalence: what share of workflow histories, and of repositories, show at least one reach drift event in the observation window?
* RQ2 Rate: what is the Reach Drift Rate?
* RQ3 Detection gap: (a) what share of drift events occur on identities that a static posture score rates below the median at the time of the event, and (b) what share of reach growth coincides with a tier increase (visible to static scoring) versus no tier change (invisible)?
* RQ4 Coverage: which NHI 2025 and Agentic 2026 categories are observable from workflow evidence, and which are not?

## 2. Definitions (draft)
* Identity unit: a workflow history, meaning one workflow file tracked across its versions and renames using the dataset's history identifier.
* Reach set at a version: distinct external capabilities the workflow references. These are third party actions and reusable workflows (identified by owner/repo[/path], ignoring the ref), secret names referenced, and environments referenced.
* Tier at a version: the declared permissions at workflow and job level, plus the `id-token` permission. A missing permissions key is recorded as "default". The real default depends on repository or organization settings and is not observable. File owner is not observable in the dataset and is excluded.
* Tier change direction: up (any permission widened or `id-token` write added), down (narrowed), or none.
* Gain: an element in the reach set that was absent in the previous version. A version only change (same owner/repo path, different ref) is not a gain. A replacement counts as one gain plus one loss, reported separately.
* Drift event: a commit by a non bot author with at least one gain and tier change direction none.
* Visible growth event: a commit with at least one gain and tier change direction up.
* Reach Drift Rate: drift events per workflow history per 90 days observed.
* Pin type per element: SHA, full tag, major tag, branch, or other. Recorded at every version.
* Bot commit: an author on a documented list of bot accounts (Dependabot, Renovate, the GitHub Actions bot, others). Bot commits are excluded from counts and reported separately.

## 3. Cohorts and data
* Cohort A (main sample): the Mons GitHub Actions workflow history dataset (Rostami Mazrae, Decan, Mens, Wessel, arXiv 2602.14572). Dataset doi 10.5281/zenodo.17301952, replication package doi 10.5281/zenodo.18414913. It holds commit level histories and file contents for public repositories with at least 300 stars and 300 commits, to 25 August 2025.
* Inclusion: valid YAML history, at least 2 versions, history existing at the start of the window.
* Observation window: the last 12 months of the dataset.
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
Public data only. Credit and comply with the dataset licence (to be checked). Release derived tables, not file bodies. Aggregate reporting, with no repositories named in negative findings. If a live secret is seen during review, do not store it and notify the owner.

## 8. Threats to validity
* Config is a proxy for runtime reach.
* The dataset covers popular, active repositories only, so results may not generalize to small projects.
* Primary branch only, and squash merges can hide separate changes.
* Histories with invalid YAML are excluded, as in the source study.
* Reach can change with no commit when a reference is a mutable tag or branch. We record pin type and report the share of reach under mutable references, but do not count such changes as drift.
* A missing permissions key hides the real default token permissions.
* Bot detection by author name is imperfect.
* Workflows are automation identities, not all agents. Cohort A supports the NHI claim. Cohort B supports the agent claim only weakly.

## 9. Related work (preliminary, not systematic)
* Rostami Mazrae et al., arXiv 2602.14572: workflow evolution at scale, permission additions, version pinning trends, bot updates. No reach versus tier framing.
* Vendor analysis of the top 2,000 repositories reporting overly permissive workflows: a static snapshot.
* Hush Security report on about 82,000 public MCP configs: credential prevalence, vendor report.
* Studies of Actions marketplace evolution and dependency rug pulls (SecDev 2025, ICSA 2026): read from abstracts only.
* A systematic search with logged terms and databases is still to be done before preregistration.

## 10. Preregistration and open decisions
Sequence: pilot, freeze definitions, preregister (for example on OSF), then the full run.
Decided: reuse the Mons dataset for Cohort A. Cohort B comes later from our own crawl. Cohort A is the main sample.
Open:
* Dataset licence and the exact fields available
* Definition alignment with rpmsft9, including what "tier" means in nhi-scan
* Identity of the second rater
* Systematic literature search

## 11. Decision log
* 23 November 2026: reuse the Mons dataset for Cohort A, to save collection time and improve reproducibility. Accepted cost: popularity bias.
* 23 November 2026: exclude bot version bumps and define gain by owner/repo path, since bots account for most version updates in the source study.
