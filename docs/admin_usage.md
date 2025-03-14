# For admin users

As an Goliac admin user, you want to be able to adapt the Goliac configuration to your needs. Usually via the `goliac.yaml` file. but not only

## golden reviewers

If you have a power users team that you want to be able to approve PRs in any repository, you can define a `golden_reviewers` team in the `goliac.yaml` file:

```yaml
admin_team: ...

golden_reviewers:
  - all-pm-team
  - principal-architects

...
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
