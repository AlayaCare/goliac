# How to install and use

You need to
- create the Goliac Github App (and points to the server)
- create the IAC github repository (with the Github actions associated) or clone the example repository
- deploy the server somewhere (and configured it)

## Creating the Goliac Githun App

In Github
- Register new GitHub App
  - in your profile settings, go to `Developer settings`/`Github Apps`
  - Click on `New Github App`
- Give basic information:
  - Github App  name can be `goliac-project-app` (it will be used in the rulesets later)
  - Homepage URL can be `https://github.com/Alayacare/goliac`
  - Disable the active Webhook
- Under Organization permissions
  - Give Read/Write access to `Administration`
  - Give Read/Write access to `Members`
- Under Repository permissions
  - Give Read/Write access to `Administration` 
  - Give Read/Write access to `Repository Content` 
- Where can this GitHub App be installed: `Only on this account`
- And Create
- then you must
  - collect the AppID
  - Generate (and collect) a private key (file)
- Go to the left tab "Install App"
  - Click on "Install"

## The goliac application

By using the docker container (`ghcr.io/nzin/goliac`) or the standalone application, goliac comes with different commands

| Command  | Description                                                                    |
|----------|--------------------------------------------------------------------------------|
| scaffold | help you bootstrap an IAC structure, based on your current Github organization |
| verify   | check the validity of a local IAC structure. Used for the CI (for example)  to valiate a PR |
| plan     | download a teams IAC repository, and show changes to apply                     |
| apply    | download a teams IAC repository, and apply it to Github                        |
| serve    | starts a server (and a UI) and apply automaticall every 10 minutes             |
| syncusers| get the definition of users outside and put it back to the IAC structure       |


## Creating the IAC github repository

In your Github organization, you need to create a git repository. Usually it is called `teams`.

You have different way to initialize it.

### Manual initialization

You can check https://github.com/goliac-project/teams

You need the following structure:
```
/
├─ goliac.yaml
├─ rulesets/
│  ├─ <rulesetname>.yaml
│  ...
├─ archived/
├─ users/
│ ├─ org/
│ │ ├─ <user>.yaml
│ │ ...
│ └─ protected/
│   ├─ <user>.yaml
│   ...
└─ teams/
   ├─ <teamname>/
   │ ├─ team.yaml
   │ ├─ <reponame>.yaml
   │ ├─ <reponame>.yaml
   │  ...
   ├─ <teaname>/
   │ └─ team.yaml
   │ ...
   ...
```

### Assisted initialization

You can use the goliac application to assist you:

```
export GOLIAC_GITHUB_APP_ID=<appid>
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=<private key filename>
export GOLIAC_GITHUB_APP_ORGANIZATION=<your github organization>
./goliac scaffold <directory> <existing github admin team name in your organization>
```

So something like
```
mkdir teams
export GOLIAC_GITHUB_APP_ID=355525
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=goliac-project-app.2023-07-03.private-key.pem
export GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project
./goliac scaffold teams admin
```

The application will connect to your Github organization and will try to guess
- your users
- your teams
- the repos associated with your teams

And it will create the corresponding structure

## the goliac.yaml configuration file

To make Goliac working you can configure the `/goliac.yaml` file

```
admin_team: admin # the name of the team (in the `/teams` directory ) that can admin this repository 
everyone_team_enabled: false # if you want all members to have read access to all repositories

rulesets:
  - pattern: .*
    ruleset: default

max_changesets: 50 # protection measure: how many changes Goliac can do at once before considering that suspicious 
archive_on_delete: true # dont delete directly repository, but archive them first

destructive_operations:
  repositories: false # can Goliac remove repositories not listed in this repository 
  teams: false        # can Goliac remove teams not listed in this repository
  users: false        # can Goliac remove users not listed in this repository
  rulesets: false     # can Goliac remove rulesets not listed in this repository
```

and you can configure different ruleset in the `/rulesets` directory like
```
apiVersion: v1
kind: Ruleset
name: default
spec:
  enforcement: evaluate # can be disable, active or evaluate 
  bypassapps:
    - appname: goliac-project-app
      mode: always # always or pull_request
  on:
    include: 
      - "~DEFAULT_BRANCH" # it can be ~ALL,~DEFAULT_BRANCH, or branch name

  rules:
    - ruletype: pull_request # currently supported: pull_request, required_signatures,required_status_checks
      parameters:
        requiredApprovingReviewCount: 1
```

## Testing your IAC github repository

Before commiting your new structure you can use `goliac verify` to test the validity:

```
cd teams
goliac verify
```

## Applying manually

After merging your team IAC teams repository, you can begin to test and apply

```
export GOLIAC_GITHUB_APP_ID=355525
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=goliac-project-app.2023-07-03.private-key.pem
export GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project
export GOLIAC_SERVER_GIT_REPOSITORY=https://github.com/goliac-project/teams

./goliac plan https://github.com/goliac-project/teams main
```

and you can apply the change "manaully"

```
export GOLIAC_GITHUB_APP_ID=355525
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=goliac-project-app.2023-07-03.private-key.pem
export GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project
export GOLIAC_SERVER_GIT_REPOSITORY=https://github.com/goliac-project/teams

./goliac plan https://github.com/goliac-project/teams apply
```

If it works for you, you can put in place the goliac service to fetch and apply automatically (like every 10 minute). See below

## Configure the Goliac server

You can run the goliac server as a service or a docker container. It needs several environment variables:

| Environment variable             | Default     | Description                 |
|----------------------------------|-------------|-----------------------------|
| GOLIAC_LOGRUS_LEVEL              | info        | debug,info,warning or error |
| GOLIAC_LOGRUS_FORMAT             | text        | text or json                |
| GOLIAC_GITHUB_SERVER             | https://api.github.com |                  |
| GOLIAC_GITHUB_APP_ORGANIZATION   |             | name of your github org     |
| GOLIAC_GITHUB_APP_ID             |             | app id of Goliac Github App |
| GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE |           | path to private key       |
| GOLIAC_EMAIL                     | goliac@alayacare.com | author name used by Goliac to commit (Codeowners) |
| GOLIAC_GITHUB_CONCURRENT_THREADS | 1           | You can increase, like '4' |
| GOLIAC_GITHUB_CACHE_TTL          |  86400      | Github remote cache seconds retention |
| GOLIAC_SERVER_APPLY_INTERVAL     | 600         | How often (seconds) Goliac try to apply |
| GOLIAC_SERVER_GIT_REPOSITORY     |             | teams repo name in your organization |
| GOLIAC_SERVER_GIT_BRANCH         | main        | teams repo default branch name to use |
| GOLIAC_SERVER_HOST               |localhost    | useful to put it to `0.0.0.0` |
| GOLIAC_SERVER_PORT               | 18000       |                            |
| GOLIAC_SERVER_GIT_BRANCH_PROTECTION_REQUIRED_CHECK | validate | ci check to enforce when evaluating a PR (used for CI mode) |
then you just need to start it with

```
./goliac serve
```

You can connect (eventually) to the UI for some statistic to `http://GOLIAC_SERVER_HOST:GOLIAC_SERVER_PORT`

### Using docker container

```
docker run -ti -v `pwd`/goliac-project-app.2023-07-03.private-key.pem:/app/private-key.pem -e GOLIAC_GITHUB_APP_ID=355525 -e GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=/app/private-key.pem -e GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project -e GOLIAC_SERVER_GIT_REPOSITORY=https://github.com/goliac-project/teams -e GOLIAC_SERVER_HOST=0.0.0.0 -p 18000:18000 ghcr.io/nzin/goliac serve
```

## Syncing Users from an external source

You can create/edit all your users manually in the `users/org/` directory. But often you are already managing your users from another source of thruth.

Goliac can sync users from an external source, This is the `usersync` section in the `goliac.yaml` file. There are different plugins:
 
| Plugin name    | Description                                                               |
|----------------|---------------------------------------------------------------------------|
| noop           | Doing nothing (if you dont want to sync from an external source of truth) |
| fromgithubsaml | If you are using Github Enterprise SAML integration                       |
| shellscript    | If you want an ad-hoc sync method, Goliac call the `usersync.path`        |

What you need to do:
- edit the `goliac.yaml` file to specify the right `usersync` plugin
- run regularly the `./goliac syncusers` command (cronjob or k8s cronjob) to sync users definition

### Protected users

On top of syncing users, if you fear to loose control on users, or you want to ensure that some users are not deleted, you can copy their definition into the `org/protected` directory.

As a reminder a user is defined via a yaml file like `alice.yaml` with the content:

```
apiVersion: v1
kind: User
name: alice
spec:
  githubID: alice-myorg
```
