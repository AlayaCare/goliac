# Ruleset

You can create rules on your repositories:
- repository branch protection
- ruleset branch protection
- or organization wide ruleset

This page is about organization wide ruleset. Usually it is done by the Security/IT team

## Create Organization Ruleset

You can create a ruleset file in the `/rulesets` directory, like `/rulesets/default.yaml`:

```yaml
apiVersion: v1
kind: Ruleset
name: default
spec:
  repositories:
    included:
      - ~ALL
    # except:
    #   - foo
    #   - bar.*
  ruleset:
    enforcement: evaluate # can be disable, active or evaluate
    bypassapps:
        - appname: goliac-project-app
        mode: always # can be always or pull_request
#    bypassteams:
#      - teamname: GoldenReviewers
#        mode: pull_request # can be always or pull_request
    conditions:
        include: 
        - "~DEFAULT_BRANCH" # it can be ~ALL,~DEFAULT_BRANCH, or branch name
    rules:
        - ruletype: pull_request # currently supported: pull_request, required_signatures,required_status_checks, creation, update, deletion, non_fast_forward, required_linear_history
        parameters:
            requiredApprovingReviewCount: 1
```


- update the `/goliac.yaml` file to include the new organization ruleset:

```yaml
...
rulesets:
  - default
```

- the name (here `default`), is the name of the file in the `/rulesets` directory

## Repositories section

You can define which repositories will be impacted (using regular expressions) with the `included` and `except`:

```yaml
  repositories:
    included:
      # - ~ALL
      - .*
      # - prefix-.*
    except:
      - foo
      - bar.*
```

## Bypass section

You can define a application to be able to bypass the above rules:

```yaml
  ruleset:
    bypassapps:
      - appname: goliac-project-app
        mode: always # always, pull_request
```

or you can define a team that can bypass, like a "golden reviewer" team:

```yaml
  ruleset:
    bypassapps:
      - appname: alayacare-goliac # the name of your Github App
        mode: always
    bypassteams:
      - teamname: GoldenReviewers
        mode: pull_request # it can be always or pull_request
```

## Rule section

Few rules are currently supported (but the software can be easily extended): `pull_request`, `required_signatures`, `required_status_checks`, `creation`, `update`, `deletion`, `required_linear_history`

### pull_request

Require all commits be made to a non-target branch and submitted via a pull request before they can be merged

```yaml
  ruleset:
    rules:
      - ruletype: pull_request
        parameters:
          # dismissStaleReviewsOnPush: false
          # requireCodeOwnerReview: false
          requiredApprovingReviewCount: 1
          # requiredReviewThreadResolution: false
          # requireLastPushApproval: false
```

### required_signatures

Require signed commits: Commits pushed to matching refs must have verified signatures

```yaml
  ruleset:
    rules:
      - ruletype: required_signatures
```

### required_status_checks

Choose which status checks must pass before the ref is updated. When enabled, commits must first be pushed to another ref where the checks pass

```yaml
  ruleset:
    rules:
      - ruletype: required_status_checks
        parameters:
          requiredStatusChecks:
            - nameofYourStatusCheck
          # strictRequiredStatusChecksPolicy: false
```

### creation

Restrict creations: only allow users with bypass permission to create matching refs

```yaml
  ruleset:
    rules:
      - ruletype: creation
```

### update

Restrict updates: Only allow users with bypass permission to update matching refs

```yaml
  ruleset:
    rules:
      - ruletype: update
```

### deletion

Restrict deletions: Only allow users with bypass permissions to delete matching refs

```yaml
  ruleset:
    rules:
      - ruletype: deletion
```

### required_linear_history

Require linear history: Prevent merge commits from being pushed to matching refs

```yaml
  ruleset:
    rules:
      - ruletype: required_linear_history
```
