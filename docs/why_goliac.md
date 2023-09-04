# Why Goliac

Goliac is a tool to help you manage your Github Organization (repos/teams/users) in a friendly way
- for your security team (enforcing some security rules globally, reducing the number of Github adminstrators, and passing compliance audits)
- for your developpers (it is a developer self-serve tool)
- without having to rely on your IT departement each time a team needs a new repository

## Security friendly

Goliac allows your company to pass security compliance audit by:
- defining global rules (based on [Github Rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) if you are on Github Enterprise plan)
- allowing users to manage their team and the repositories they own (and only them)
- bringing auditing of who has done what in 2 places:
  - via a git history of a git repository
  - via logs of Goliac service
- via a [GitOps](https://www.redhat.com/en/topics/devops/what-is-gitops) approach: i.e. via a clear directory structure stored into a git repository

## Developer friendly

Once a team of developer has been created, the team can
- manage their resources (i.e. their team's member and their repositories defintion) autonomously
- via **simple** yaml files. You dont need to learn a new specific definition langage.
- but restricted by global policies defined previously by the security team. For example you can specifiy a organization-wide policy asking for peer-review across all Github repositories, before any Pull Request being merged. Or you can ask a specific CI test to pass for all Github repositories, or a specific subset of Github repositoties

## How it works

Goliac use a [GitOps](https://www.redhat.com/en/topics/devops/what-is-gitops) approach:
- you define into one Github repository (usually called `teams`),through yaml files, organized into a file hiearchry, the state you want your Github organization to be. You define
  - your security rules
  - your users (or you import/link them from another external source)
  - your teams
  - the repositories owned by each teams
- when Goliac runs, it will apply these state to your Github organization (and enforce it)
- each change you want to bring is done via a Github Pull Request, that needs to be reviewed and validated, and can be auditing via the Git commit history

## Why not using other tools?

There are several existing tools that can help you define automatically your Github organization

### Why not using terraform/another tool

You can use Terraform (and a git repository) to achieve almost the same result, except that with terraform, you still need to centrally managed all operations via your IT team.

Goliac allows you to provide a self-served tool to all your employees

### Why not using Github integrations

Github itself allows you different integrations (see `https://github.com/ORG/goliac-project/settings/security`), in particular 
- SSO users integration (SAML)
- and teams integration (Azure Active Directory and Okta)

If Github integration fits your needs, go for it. But if you want some flexibility (creating teams independently than the company organization, not having to rely on the IT departement each time you want to create a repository), Goliac may be more flexible to use.

## Requirements

To use the full capabilities of Goliac, through the Github Rulesets features you need
- either to be on Github cloud on the Enteprise plan
- or use GHES (Github Enterprise Server) 3.11

## Cost and installation

- Goliac is a free opensource project.
- currently Goliac manasges 1 Github organization per instance
- The installation is relatively easy:
  - either you install a stanadlone application
  - or you use Goliac docker image that you can build yourself or use pre-built images (https://github.com/nzin/goliac/pkgs/container/goliac), and it has been designed to be run easily into a kubernetes environment
  - you need to create a definition of what Goliac manages: either partial or full definition of your Github organization
- the definition (and the maintenance) of the definition of your Github organization is done via simple yaml file, and dont requires special skills or langage know-how
