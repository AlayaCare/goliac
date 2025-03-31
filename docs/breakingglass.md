# Breaking Glass workflow

![PR breaking glass - wizard](images/breakingglass.png)

![PR breaking glass - wizard 2](images/forcemerge_workflow.png)

You have the possibliity to enable a Breaking Glass workflow that allows to force merge PRs in specific repositories and for specific teams. This is useful when you need to merge a PR in an emergency situation.

![PR breaking glass - merged](images/forcemerge_pr_merged.png)


## Register the Github App

To be able to enable the Breaking Glass workflow, you need to register the Github Appwith OAuth permissions. To do so

- go to the Github App settings (like https://github.com/organizations/AlayaCare/settings/apps/alayacare-goliac)
- you need to create a client secret (if you don't have one already)
- in General, under Identifying and authorizing users
    - set the Callback URL to `https://<Goliac DNS endpoint>/api/v1/auth/callback`
- and you need to set the following (new) environment variables:
  - `GOLIAC_GITHUB_APP_CLIENT_SECRET` (the secret associated with the webhook)
  - `GOLIAC_GITHUB_APP_CALLBACK_URL` (the `Callback URL` of your Github App)



## Enable the Breaking Glass workflow

To enable the Breaking Glass workflow, you need to
- create (or several) `/workflows/_afile_.yaml`:

```yaml
apiVersion: v1
kind: Workflow
name: _afile_
spec:
  description: General breaking glass PR merge workflow
  workflow_type: forcemerge
  repositories:
    allowed:
      - .* # you can use ~ALL
    # except:
    #   - .*-private
  acls:
    allowed:
      - team4
    #   - otherteam.*
    #   - ~ALL
    # except:
    #   - team1
  steps: # optional step to execute before force merging the PR
    - name: jira_ticket_creation
      properties:
        project_key: SRE
        issue_type: Bug
    - name: slack_notification
      properties:
        channel: sre
```

- update the `/goliac.yaml` file to include the new workflow:

```yaml
...
workflows:
- _afile_
```

## Use the Jira step

![Jira PR breaking glass](images/forcemerge_jira_ticket.png)

The Jira step is optional and can be used to create a Jira issue before force merging the PR. The step is defined as follows:

```yaml
steps:
  - name: jira_ticket_creation
    properties:
      project_key: SRE
      issue_type: Bug
```

You will need to set the following environment variables:
- `GOLIAC_WORKFLOW_JIRA_ATLASSIAN_DOMAIN` like `mycompany.atlassian.net` or `https://mycompany.atlassian.net`
- `GOLIAC_WORKFLOW_JIRA_EMAIL` of the service account
- `GOLIAC_WORKFLOW_JIRA_API_TOKEN` of the service account


## Use the Slack step

The Slack step is optional and can be used to send a notification to a Slack channel before force merging the PR. The step is defined as follows:

```yaml
steps:
  - name: slack_notification
    properties:
      channel: sre
```

You will need to set the following environment variables:
- `GOLIAC_SLACK_TOKEN` the Slack token
- `GOLIAC_SLACK_CHANNEL` the Slack channel

Dont forget to invite the Slack bot to the channel.

