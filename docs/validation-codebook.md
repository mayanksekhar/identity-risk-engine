# Validation codebook, version 1.0 (23 November 2026)

Applies to protocol section 6. Two raters label each item independently, without
seeing the other rater's sheet and without being told whether the detector
flagged the item.

## The question
Did this commit introduce a genuinely new third party dependency into this
workflow? Third party means an action or reusable workflow whose owner is not
actions or github. Judge the reach change only. Ignore the permission tier.

## Labels (column label)
- true_reach_gain: the commit makes the workflow depend on a third party action
  or reusable workflow it did not depend on before. A swap to a different
  project counts, even if an old action was removed in the same commit.
- false_positive: the detector's gain is not a new third party dependency.
  Choose exactly one reason in the fp_reason column.
- ambiguous: you cannot decide after about three minutes. Say why in notes.

## False positive reasons (column fp_reason)
Apply in this order and stop at the first that fits.
1. FP_SELF: the added action belongs to the same GitHub user or organization as
   the repository, or is a path inside the same repository.
2. FP_PARSE: the detector misread the file. Examples: text in a comment or a
   non step key, an expression, or a path that is not an action.
3. FP_RENAME: the same project under a new owner or repository name (rename,
   transfer, or a fork replacing its upstream by the same maintainers).
4. FP_MOVED: the action already ran elsewhere in the same repository and this
   commit moved it between workflow files. Check the commit's other files.
5. FP_REVERT: the commit restores an action that an earlier commit removed from
   this workflow. Check the file history.
6. FP_OTHER: anything else, with a short explanation in notes.
If none of FP_SELF to FP_OTHER fits, the label is true_reach_gain.

## Unflagged items
The same question applies. If you find a new third party dependency the added
list does not show, label true_reach_gain and fill reach_type with one of:
action, reusable_workflow, docker_image, script_fetch, other.

## Procedure
Spend at most three minutes per item. You may open the commit and file history
on GitHub using the repository and commit_hash columns. Do not discuss items
with the other rater until both sheets are complete.

## Agreement and adjudication
Cohen's kappa is computed on the three labels (true_reach_gain, false_positive,
ambiguous) before any discussion. Disagreements are then resolved by discussion
and both the pre and post adjudication results are reported. Rater identities
and the date range of labeling are recorded in results/.
