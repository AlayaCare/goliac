# User

Under `/users`, you find the deifinion of your Github users.

There are 3 different sub directories
- `/users/org` where you find all your organization users
- `/users/protected` where you find all your organization users but you want to protect them to be removed by the github SAML sync. Usually you put your admin users here
- `/users/external` where you find external (to your organization) users, you want to be part of your repositories (as readers or writers)

If you dont have Github enterprise (and a SAML integration), you will need to manually create your organization users definition


## User definition

```yaml
apiVersion: v1
kind: User
name: alice
spec:
  githubID: aliceGithubUserName
```

## Automatic SAML sync

For the SAML integration check the `/goliac.yaml` [configuration file](/installation), in particular: 

```yaml
usersync:
  plugin: noop # noop, fromgithubsaml, shellscript
```

it will allow Goliac to
- sync from Github to the goliac-teams repository all users
- remove the user from teams when they are no longer part of your organization