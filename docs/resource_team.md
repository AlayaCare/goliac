# Team

Under the /teams directory, you can create one subdirectory per team.


## Create a new team

If you want to create a new team (like `foobar`), you need to create a PR with a `/teams/foobar/team.yaml` file:

```yaml
apiVersion: v1
kind: Team
name: foobar
spec:
  owners:
    - user1
    - user2
  members:
    - user3
    - user4
```

The users defined there are in 2 different categories
- members: are part of the team (and will be writer on all repositories of the team)
- owners: are part of the team (and will be writer on all repositories of the team) AND can approve PR in the `foobar` teams repository (when you want to change a team definition, or when you want to create/update a repository definition)

The users name used are the one defined in the `/users` sub directories (like `alice`)


## Externally managed team

If the definition of a team is externally managed (your IT team is responsible to push the definition of a team via a tool/script), you can set a specific property to tell Goliac to not own/enforce the definition of a team:


```yaml
apiVersion: v1
kind: Team
name: AnotherTeam
spec:
  externallyManaged: true
```

It means that Goliac will not try to enforce the team definition, but will take it as it is in Github. It will only use the team to manage the repositories.

