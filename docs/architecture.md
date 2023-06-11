# Architecture

# The IAC repo structure

```
/
  | .github/
  | \- CODEOWNERS
  | org/
    | settings.yaml
    | profiles/
    | users/
    | |- protected/
    | | |- <user1>.yaml
    | | |- <user2>.yaml
    | | ...
    | |- org/
    | | |- <user1>.yaml
    | | |- <user2>.yaml
    | | ...
    | |- external/
    |   | |- <user1>.yaml
    |   | |- <user2>.yaml
    |   | ...
    | teams/
      |<team1>/
      | |-team.yaml
      | |-<repo1>.yaml
      | ...
      |<team2>/
      | |-team.yaml
      | \-<repo1>.yaml
      | ...
    | archived-repos/
      |-<repo1>.yaml
      |-<repo2>.yaml
      | ...
```

## Org setting file

```
apiVesrion: v1
kind: OrgSettings
data:
  removeUnknownUsersFromGithub: [true|false]
  removeUnknownTeamsFromGithub: [true|false]
  removeIndividualFromRepos: [true|false]
```

## Profile file

_profilename_.yaml:
```
apiVesrion: v1
kind: Profile
metadata:
  name: <profilename>
data:
  everyoneTeam: [true|false] # creating an everyone (inside org) team that have read access to all repos
  branchNamesAllowed:
    - main
    - develop
  enforcedBranchProtection:
    - default:
      requiredNumberOfApproval: <number>
      ...
    - main:
    ...
```

## user file

_username_.yaml
```
apiVesrion: v1
kind: User
metadata:
  name: <username>
data:
  githubID: <githubid>
```

## team file

team.yaml
```
apiVesrion: v1
kind: Team
metadata:
  name: <username>
data:
  owner:
    - <username>
    - <username>
  members:
    - <username>
    - <username>
```

will create 2 github teams:
- <team>-owners
  - that can approve repo change via CODEOWNER on this github definition repo
  - that can approve PR merge into the different owned repos 
- <team>-members
  - that can approve PR merge into the different owned repos 

## repo file

_repo_.yaml
```
apiVesrion: v1
kind: Repository
data:
  writer:
    - <team>
  reader:
    - <team>
  branches:
    - main:
      enforcedBranchProtection:
        requiredNumberOfApproval: <number>
        ...
```

# Action

There are several modes
- file validation
- application

## File validation

- read users list
- stop if files are present which are not correctly formatted
- read teams and repos list
- stop if files are present which are not correctly formatted
- remove unknown users from teams
- flag (warning) teams that do not have 2 owners

## Application

- read users list on disk
- fetch user list from Github
- (orgSetting bool option) remove from Github users not in the on disk list 
- read teams list on disk
- fetch team list from Github
- (orgSetting bool option) remove from Github teams not in the on disk list 
- read repos list on disk
- fetch repos list from Github
- for each repo:
  - (orgSetting bool option) remove individuals from Github repo (we want only teams)
  - apply team's ownership (and remove extra)
  - apply repo permission

# User sync

Source information for users currently in the organization is usually coming from an external entity (from Active Directory for example).
We can provide some user sync applications for "common" external source of truth