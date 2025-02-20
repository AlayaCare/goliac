# What is Goliac

Goliac is a tool to manage your Github organization in a GitOps way, a bit like [ArgoCD](https://argoproj.github.io/argo-cd/), (or [Terraform](https://www.terraform.io/) ) but for Github organization.

It allows you to
- define your Github organization structure (teams, users, repositories) into a git repository, and to apply this structure to your Github organization.
- enforce security rules (like [Github Rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets)) across your Github organization
- allow your developers to manage their team and their repositories (and only them) autonomously

You will get the most of Goliac if you are on an Enterprise plan, or on prem (especially to be able to use [Github Ruleset](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets)), and it runs well for organization with ~ 1000-2000 repos, ~ 500-1000 users, ~ 300-500 teams or below. It should works well above these numbers, but you may need to adapt the way you use it (because it may need to do a lot of API calls to Github).

Goliac is a free opensource project, that you can install on your own infrastructure, and that is designed to be run easily into a kubernetes environment.

## Why Goliac

Goliac can improve your Github organization management in several ways:
- security
- developer friendly
- cost

## How it works

![goliac workflow](images/goliac_basic_workflow.png)

Goliac use a [GitOps](https://www.redhat.com/en/topics/devops/what-is-gitops) approach:
- you define into one Github repository (usually called `goliac-teams`), through yaml files, organized into a file hierarchy, the state you want your Github organization to be. You define
  - your security rules
  - your users (or you import/link them from another external source)
  - your teams
  - the repositories owned by each teams
- when Goliac runs, it will apply these state to your Github organization (and enforce it)
- each change you want to bring is done via a Github Pull Request, that needs to be reviewed and validated, and can be auditing via the Git commit history
- each team can change part of the `goliac-teams` structure they own (a sub directory)


## Security friendly

Goliac allows your company to pass security compliance audit by:
- defining global rules (based on [Github Rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) if you are on Github Enterprise plan)
- allowing users to manage their team and the repositories they own (and only them)
- bringing auditing of who has done what in 2 places:
  - via a git history of the `goliac-teams` repository
  - via logs of Goliac service (but you need to have a good log management system in place)
- via a [GitOps](https://www.redhat.com/en/topics/devops/what-is-gitops) approach: i.e. via a clear directory structure stored into a git repository

## Developer friendly

Once a team of developers has been created, the team can
- manage their resources (i.e. their team's member and their repositories defintion) autonomously (and so without having to rely on your IT departement each time a team needs a change)
- via **simple** yaml files. You dont need to learn a new specific definition langage.
- but restricted by global policies defined previously by the security team. For example you can specifiy a organization-wide policy asking for peer-review across all Github repositories, before any Pull Request being merged. Or you can ask a specific CI test to pass for all Github repositories, or a specific subset of Github repositoties

## Cost

Goliac is a free opensource project. You can install it on your own infrastructure, and it is designed to be run easily into a kubernetes environment.

A comparable solution is to use Terraform (and a git repository) to achieve almost the same result, except that
- if you are using Terraform Cloud, you will have to pay for each resource you manage
- with terraform, you still need to centrally managed all operations via your IT team, which can be a bottleneck, and also less flexible


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
- currently Goliac manages 1 Github organization per instance
- The installation is relatively easy:
  - either you install a standalone application (Goliac app comes as a single binary)
  - or you deploy Goliac docker image (via docker-compose or in kubernetes). You can build yourself the docker image or use pre-built images (https://github.com/goliac-project/goliac/pkgs/container/goliac).
  - you need to create a definition of what Goliac manages: aka the goliac "teams" repository. With it you can define and managed either partially or totally your Github organization
- the definition (and the maintenance) of your Github organization is done via simple yaml file, and dont requires special skills or langage know-how
