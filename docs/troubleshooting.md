# Troubleshooting guide

## How to resolve the error "more than X changesets to apply (total of Y), this is suspicious. Aborting"

This error is happening if a changeset (a team's PR) introduce more than X changesets. This is a safety mechanism to avoid applying a huge number of changesets at once.

If it is a legitimate change, you can
- either increase the `max_changesets` in the `goliac.yaml` file, but that's not the best approach.
- create a new PR to reduce the number of changes, Goliac will automatically apply the cumulative changesets.
- or you can use the CLI to force apply the changesets. To do so, you can run the following command:

```bash
export GOLIAC_GITHUB_APP_ORGANIZATION=<your organization>
export GOLIAC_GITHUB_APP_ID=<github app id>
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=<github app private key filename>
GOLIAC_MAX_CHANGESETS_OVERRIDE=true ./goliac apply <github teams url> <branch>
```

For example:

```bash
export GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project
export GOLIAC_GITHUB_APP_ID=123456
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=github-app-private-key.pem
GOLIAC_MAX_CHANGESETS_OVERRIDE=true ./goliac apply https://github.com/goliac-project/teams main
```

Note: it is possible that Goliac will be a bit confused after your force changes. You will certainly need to restart Goliac (app/kubernetes pod) just after running this command,

## How to bypass Goliac for a specific repository

If you want to force merge a PR without Goliac validation, you will need to disable Golac for this specific repository temporarily.
To do so, as a Gitbub admin, you can go to
- the Github organization settings,
- on the left menu, under `Code planning and automation` / `Repositories`, search for `Rulesets`
- usually there is a `default` ruleset, click on it
- then under `Target repositories`, you can search for the repository you want to bypass Goliac for, unselect it
- then click on `Save changes` (at the bottom of the page)

Note:
- When Goliac will run (and its cache expires), it will put back the ruleset. Usually the cache is set to 86400 seconds (ie 1 day).
- if you want to re-apply the ruleset quickly (when you have finished with your emergency chage), you can go to the Goliac UI and click on the `Flush cache` button, and then click on the `Re-Sync` button.