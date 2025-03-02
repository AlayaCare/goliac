# Quick start for existing Github organization

For the full installation documentation, check the [installation](installation.md) page

## Preparation

### You need a Github app

As a Github org admin, in GitHub:
- Register new GitHub App
  - in your profile settings, go to `Developer settings`/`GitHub Apps`
  - Click on `New GitHub App`
- Give basic information:
  - GitHub App  name can be `<yourorg>-goliac-app` (it will be used in the rulesets later)
  - Homepage URL can be `https://github.com/goliac-project/goliac`
  - Disable the active Webhook
- Under Organization permissions
  - Give Read/Write access to `Administration`
  - Give Read/Write access to `Members`
- Under Repository permissions
  - Give Read/Write access to `Administration`
  - Give Read/Write access to `Content`
- Where can this GitHub App be installed: `Only on this account`
- And Create
- then you must
  - collect the AppID
  - Generate (and collect) a private key (file)
- Go to the left tab "Install App"
  - Click on "Install"

#### Alternative: use a personal access token

If you don't have the possibility to create a Github App, you can use a personal access token.
If you only need to scaffold, you will need a personal access token with
- `read:org` scope (under `admin:org` category)

If you want to use the full Goliac capabilities, you will need a personal access token with
- `read:org` and `write:org` scope (under `admin:org` category)

You will need to export the `GOLIAC_GITHUB_PERSONAL_ACCESS_TOKEN` env variable (instead of `GOLIAC_GITHUB_APP_ID`, `GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE`) in the following examples.

### Get the Goliac binary

```shell
curl -o goliac -L https://github.com/goliac-project/goliac/releases/latest/download/goliac-`uname -s`-`uname -m` && chmod +x goliac
```

## Starting with Goliac

You can onboard your repositories and teams incrementally. You can start with a single team and a single repository and then add more as you go.

The best way to do it is to:
- create a `goliac-admin` dedicated Github team to administer Goliac
- use the `goliac scaffold` command to create the initial structure and then cherry-pick the repositories and teams you want to onboard.
- optionally use the `goliac verify` command to check that the structure is correct (the scaffold command is supposed to do it for you)
- use the `goliac plan` command to see what Goliac will do
- use the `goliac apply` command to apply the changes or merge the PRs


### Create a goliac admin team

If you dont have yet one, you will need to create a team in Github, where you will add your IT/Github admins (in our example, the team is called `goliac-admin` ), that will administer Goliac.

### Scaffold

And now you can use the goliac application to assist you:

```shell
export GOLIAC_GITHUB_APP_ID=<appid>
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=<private key filename>
export GOLIAC_GITHUB_APP_ORGANIZATION=<your github organization>
./goliac scaffold <directory> <goliac-admin team you want to create>
```

So something like

```shell
mkdir goliac-teams
export GOLIAC_GITHUB_APP_ID=355525
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=goliac-project-app.2023-07-03.private-key.pem
export GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project
./goliac scaffold goliac-teams goliac-admin
```

The application will connect to your GitHub organization and will try to guess
- your users
- your teams
- the repos associated with your teams

And it will create the corresponding structure into the "goliac-teams" directory.

In particular it will creates a `/goliac.yaml` file:

```yaml
admin_team: goliac-admin

rulesets:
  - pattern: .*
    ruleset: default

max_changesets: 50
archive_on_delete: true

destructive_operations:
  repositories: false
  teams: false
  users: false
  rulesets: false

usersync:
  plugin: fromgithubsaml
```

This default behaviour
- forbids any destructive operations
- uses the `fromgithubsaml` plugin to sync users (for Enterprise GitHub plan)
- dont force you to onboard all repositories and teams at once (i.e. you can do it incrementally)
- uses a global ruleset called `default` for all repositories (check the `rulesets/default.yaml` file)


## Starting the onboarding

If you want to start simple, in the `teams` repository you can remove all the teams (except one) and repositories (except one), to have something like

- teams/myteam/team.yaml

```yaml
apiVersion: v1
kind: Team
name: myteam
spec:
  owners:
    - user1
  members:
    - user2
```

- teams/myteam/myrepository.yaml

```yaml
apiVersion: v1
kind: Repository
name: myrepository
...
```

### Verify

Eventually you can check the structure of your configuration, by running 

```shell
goliac verify <goliac-team directory>
```

### Publish the goliac teams repo

In Github create a new repository called (for example) `goliac-teams`.

And commit the new structure:

```
cd goliac-teams
git init
git add .
git commit -m "Initial commit"
git push
```

### Adjust (i.e. plan)

Run as much as you want, and check what Goliac will do

```shell
export GOLIAC_GITHUB_APP_ID=355525
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=goliac-project-app.2023-07-03.private-key.pem
export GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project
./goliac plan --repository https://github.com/goliac-project/goliac-teams --branch main
```

### Apply

If you are happy with the new structure:

```shell
export GOLIAC_GITHUB_APP_ID=355525
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=goliac-project-app.2023-07-03.private-key.pem
export GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project
./goliac apply --repository https://github.com/goliac-project/goliac-teams --branch main
```

### Run the server

You can run it locally

```shell
export GOLIAC_GITHUB_APP_ID=355525
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=goliac-project-app.2023-07-03.private-key.pem
export GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project
export GOLIAC_SERVER_GIT_REPOSITORY=https://github.com/goliac-project/goliac-teams
#export GOLIAC_SERVER_GIT_BRANCH=main # by default it is main

./goliac serve
```

And you can access the dashboard UI at http://localhost:18000

## Incremental onboarding

You can now add more teams and repositories, by running the `scaffold` command, and cherry-pick more files and put them in the `goliac-teams` repository:
- create a new branch
- add the new team/repository
- verify
- push the branch
- plan (with the name of the branch)
- merge the branch

## Use daily

### Add repositories

Now everyone can enroll existing (or new) repositories.

Let's imagine you want to control the `myrepository` repository for the existing team `ateam`, one of the `ateam` member (or you) can
- create a new branch into the `goliac-teams` repository
- add the repository definition
- push and create a PullRequest
- one of the team "owner" or one of the `goliac-admin` member will be able to approve and merge

```shell
git checkout -b addingRepository

cd goliac-teams/teams/ateam

cat >> myrepository.yaml << EOF
apiVersion: v1
kind: Repository
name: myrepository
spec:
  visibility: private
EOF

git add myrepository.yaml
git commit -m 'adding myrepository under Goliac'
git push origin addingRepository
```

and go to the URL given by Github to create your Pull Request

