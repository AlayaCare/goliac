# Breaking Glass workflow

If you enable the Breaking Glass workflow, you will be able to force merge PRs in specific repositories and for specific teams. This is useful when you need to merge a PR in an emergency situation.

## First way: use the wizard

![PR breaking glass - wizard](images/breakingglass.png)

![PR breaking glass - wizard 2](images/forcemerge_workflow.png)

You have the possibliity to enable a Breaking Glass workflow that allows to force merge PRs in specific repositories and for specific teams. This is useful when you need to merge a PR in an emergency situation.

![PR breaking glass - merged](images/forcemerge_pr_merged.png)


See [Workflows](resource_workflow) to see how to enable it

## Second way: use PR comments

You can also use PR comments to force merge a PR.

```
/forcemerge:<workflow_name>: <explanation>
```

Example:

```
/forcemerge:standard: need to merge this PR out of regular office hours to fix the production issue
```

This will trigger the `standard` workflow to merge the PR.

## Squash merge

By default, the PR will be merged with the `commit` merge type.


If you want to squash merge the PR, you can add the `/squash` string in the explanation.

Example:

```
/forcemerge:standard: need to merge this PR out of regular office hours to fix the production issue /squash
```

This will trigger the `standard` workflow to merge the PR with the `squash` merge type.

## Stacked PRs

GitHub only merges [stacked pull requests](https://docs.github.com/en/rest/pulls/pulls#merge-a-pull-request-asynchronously) through its asynchronous merge API. For a PR that is part of a stack, you can add an optional merge mode between the workflow name and the explanation:

```
/forcemerge:<workflow_name>: <explanation>
/forcemerge:<workflow_name>:direct: <explanation>
/forcemerge:<workflow_name>:stack: <explanation>
```

The merge mode is the token right after the workflow name colon (case-insensitive): `/forcemerge:standard:stack: why`, or a bare `/forcemerge:standard:stack`. After a space it is explanation text, so `/forcemerge:standard: stack: why` and `/forcemerge:standard: fix: prod is down` have no mode. A word immediately followed by a colon that is not `stack` or `direct` (`/forcemerge:standard:fix: x`) is refused as an unknown merge mode. The approval review always states the resolved mode. A redelivered GitHub comment (same comment id) is ignored, and a second force merge of a PR that is already being force merged is refused.

Take the stack `1 > 2 > 3 > 4 > main`: PR 1 is the top (based on PR 2's branch) and PR 4 is the bottom (based on `main`).

- **Default, or `direct`**: merge only this PR into its base branch, then restack.
  - `/forcemerge:standard: <explanation>` on PR 3 merges PR 3 into PR 4's branch with a merge commit. Then every open PR based on PR 3's branch (PR 2) is retargeted onto PR 4's branch. The stack becomes `1 > 2 > 4 > main`.
  - On the bottom PR (PR 4), the PR is merged alone into `main` through the asynchronous merge API, and PR 3 is retargeted onto `main`.
  - On the top PR (PR 1), the PR is merged into PR 2's branch and there is nothing to retarget.
  - `/squash` is not available when merging into a parent branch: a merge commit is used, and the approval comment says so.
  - GitHub usually marks the PR as merged once its commits are in its base branch. If it does not, Goliac closes the PR with a comment saying where it was merged.
  - The PR's head branch is not deleted.
- Before anything is approved, Goliac checks the PR (and, for `stack`, every PR of the stack): open, not merged, not a draft, head in the repository (no fork), consistent stack metadata, and for a direct merge into a parent branch that the PR below exists in the same stack. Otherwise nothing is approved, merged or notified. The workflow steps (audit, Slack...) run once per merged PR after this check and before the first change.
- The merge method is chosen from the repository settings (`allow_merge_commit`, `allow_squash_merge`, `allow_rebase_merge`): the default merge commit becomes a squash (then a rebase) when the repository forbids merge commits (e.g. squash-only repositories). `/squash` is never replaced: it is refused when squash is not allowed. Merging a mid-stack PR into its parent branch always makes a merge commit (it is a branch merge, not a PR merge, so it does not depend on those settings); on a squash-only repository that commit lands on the parent PR's branch and the bottom PR is squashed into the default branch afterwards.
- A PR is retargeted only after GitHub confirms it merged (`merged: true`). `enqueued` (merge queue) is reported as "queued, not merged": nothing is retargeted. A foreign pending merge request (409) is reported, not adopted. Retargeting is retried with a backoff, and a persistent failure names the PRs and the manual fix.
- If the merge definitively fails after the approvals, the approvals Goliac created are dismissed (or listed in the error when they can't be). When the outcome is unknown (timeout, queue), they are left in place and listed.
- Just before the asynchronous merge the heads are re-read: a push after the approval aborts the merge.
- **`stack`**: merge the whole stack into the stack base branch.
  - `/forcemerge:standard:stack: <explanation>` on any PR of the stack approves every PR from 1 to 4. It then merges the top PR (PR 1) through the asynchronous merge API, which merges PR 1 to PR 4 into `main`.
  - If a PR of the stack is missing or not open, nothing is approved or merged.

On a PR that is not part of a stack, `direct` and `stack` are ignored (the approval comment says so) and the PR is merged as usual.

The asynchronous merge:
- pins the PR head that Goliac approved (a push in between cancels the merge);
- never uses a merge queue;
- bypasses the rules Goliac is allowed to bypass.

Goliac waits up to 2 minutes for the result. A timeout, a refusal (e.g. conflicts) or a PR that is not ready to merge (closed, draft) is reported as an error comment.
