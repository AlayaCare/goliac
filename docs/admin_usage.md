# For admin users

As an Goliac admin user, you want to be able to adapt the Goliac configuration to your needs. Usually via the `goliac.yaml` file. but not only

## Global rulesets

You can define a set of rules that will be applied to all repositories. For example, you can define that all repositories must enforce Pull Request review before being merged, to do so, you need to

1. create a file in the `rulesets` directory (for example `rulesets/default.yaml`)
2. list it in the `goliac.yaml` file

For example 

```yaml
apiVersion: v1
kind: Ruleset
name: default
spec:
  enforcement: active # active, evaluate or disabled
  bypassapps:
    - appname: alayacare-goliac # the name of your Github App
      mode: always
  conditions:
    include: 
    - "~DEFAULT_BRANCH"
  rules:
    - ruletype: pull_request
      parameters:
        requiredApprovingReviewCount: 1
```

And in the `goliac.yaml` file

```yaml
...
rulesets:
  - pattern: .* # to apply to all repositories managed by Goliac
    ruleset: default # the name of the ruleset file without the `.yaml` extension
...
```

### Define a golden reviewer team

You can define a team that can bypass this Pull Requests rule. You can change the above rule with


```yaml
apiVersion: v1
kind: Ruleset
name: default
spec:
  enforcement: active
  bypassapps:
    - appname: alayacare-goliac # the name of your Github App
      mode: always
  bypassteams:
    - teamname: GoldenReviewers
      mode: pull_request # it can be always or pull_request
  conditions:
    include: 
    - "~DEFAULT_BRANCH"
  rules:
    - ruletype: pull_request
      parameters:
        requiredApprovingReviewCount: 1
```

## externally managed teams

Something a bit more specific: if you have teams that are managed outside of Goliac, you can define a team with a specific `externallyManaged` flag:

```yaml
apiVersion: v1
kind: Team
name: AnotherTeam
spec:
  externallyManaged: true
```

It means that Goliac will not try to enforce the team definition, but will take it as it is in Github. It will only use the team to manage the repositories.
