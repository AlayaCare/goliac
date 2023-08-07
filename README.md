# ![Goliac](docs/logo_small.png) Goliac

Goliac (Github Organization Leveraged by Infrastructure As Code), is a tool to manage your Github Organization (users/teams/repositories) via
- yaml manifests files structured in a Github repository
- this IAC Github repositories can be updated by teams from your organization, but only the repositories they owns
- all repositories rules are enforced via a central configuration that only the IT/security team can update (if you are using Github Enterprice)
- a Github App watching this repository and applying any changes

## Why not using terraform/another tool

You can use Terraform to achieve almost the same result, except that with terraform, you still need to centrally managed all operations via your IT team.

Goliac allows you to provide a self-served tool to all your employees

## Why not using Github integrations

Github itself allows you different integrations (see `https://github.com/ORG/goliac-project/settings/security`), in particular 
- SSO users integration (SAML)
- and teams integration (Azure Active Directory and Okta)

If Github integration fits your needs, go for it. But if you want some flexibility (creating teams independently than the company organization, not having to rely on the IT departement each time you want to create a repository), Goliac may be more flexible to use.

## How to install and use

You need to
- install the Goliac Github App (and points to the server)
- create the IAC github repository (with the Github actions associated) or clone the example repository
- deploy the server somewhere (and configured it)

## Creating the Goliac Githun App

- Register new GitHub App
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
  - Generate (and collect) a client secret
- Go to the left tab "Install App"
  - Click on "Install"

## Creating the IAC github repository

You can check https://github.com/goliac-project/teams

You need the following structure:
```
/
|- goliac.yaml
|- rulesets/
|- archived/
|- users/
| |- org/
| | |- <user>.yaml
| | ...
| \- protected/
|   |- <user>.yaml
|   ...
\- teams/
   |- <teamname>/
   | |- team.yaml
   | |- <reponame>.yaml
   | |- <reponame>.yaml
   |  ...
   |- <teaname>/
   | |- team.yaml
   | ...
   ...
```

## Usage

### Users

Usually users are imported from from an external source.
Each user is defined as a yaml file. For example `alice.yaml`:

```
apiVersion: v1
kind: User
metadata:
  name: alice
data:
  githubID: alice-myorg
```

### Create a new team

If you want to create a new team (like `foobar`), you need to create a PR with a `/teams/foobar/team.yaml` file:

```
apiVersion: v1
kind: Team
metadata:
  name: foobar
data:
  owners:
    - user1
    - user2
  members:
    - user3
    - user4
```

The users defined there are in 2 different categories
- members: are part of the team (and will be writer on all repositories of the team)
- owners: are part of the team (and will be writer on all repositories of the team) AMD can approve PR in the `foobar` teams repository (when you want to change a team definition, or when you want to create/update a repository definition)

The users name used are the one defined in the `/users` sub directories (like `alice`)

### Create a repository

On a given team subdirectory you can create a repository definition via a yaml file (like `/teams/foobar/awesome-repository.yaml`):

```
apiVersion: v1
kind: Repository
metadata:
  name: awesome-repository
```

This will create a `awesome-repository` repository under your organization, that will be 
- private by default
- writable by all owners/members of this team (in our example `foobar`)

You can of course tweak that:

```
apiVersion: v1
kind: Repository
metadata:
  name: awesome-repository
data:
  public: true
  writers:
  - anotherteamA
  - anotherteamB
  readers:
  - anotherteamC
  - anotherteamD
```

In this last example:
- the repository is now publci
- other teams have write (`anotherteamA`, `anotherteamB`) or read (`anotherteamC`, `anotherteamD`) access

### Archive a repository

You can archive a repository, by a PR that
- move the yaml repository file into the `/archived` directory
- and chage the repository definition like
```
apiVersion: v1
kind: Repository
metadata:
  name: awesome-repository
data:
  archived: true
```

