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
  - Homepage URL can be `https://github.com/Alayacare/goliac`
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

### Get the Goliac binary

```shell
curl -o goliac -L https://github.com/Alayacare/goliac/releases/latest/download/goliac-`uname -s`-`uname -m` && chmod +x goliac
```

### Create a goliac admin team

If you dont have yet one, you will need to create a team in Github, where you will add your IT/Github admins (in our example, the team is called `goliac-admin` ).

## Scaffold and test

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

And it will create the corresponding structure into the Goliac "teams" directory.

### Clean up to start

If you want, you can remove (for now) part or all repositories:

```shell
find goliac-teams/teams -name "*.yaml" ! -name "team.yaml" -print0 | xargs -0 rm
```

### Verify

You can run the 

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
mkdir teams
export GOLIAC_GITHUB_APP_ID=355525
export GOLIAC_GITHUB_APP_PRIVATE_KEY_FILE=goliac-project-app.2023-07-03.private-key.pem
export GOLIAC_GITHUB_APP_ORGANIZATION=goliac-project
./goliac plan --repository https://github.com/goliac-project/goliac-teams --branch main
```

### Apply

If you are happy with the new structure:

```shell
mkdir teams
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

## Use daily

### Add repositories

Now everyone can enroll existing (or new) repositories.

Let's imagine you want to control the `myrepository` repository for the existing team `ateam`, one of the `ateam` member (or you) can
- create a new branch into the `goiac-teams` repository
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
  public: false
EOF

git add myrepository.yaml
git commit -m 'adding myrepository under Goliac'
git push origin addingRepository
```

and go to the URL given by Github to create your Pull Request

