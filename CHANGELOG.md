## Goliac v0.15.8

- default branch bugfixes: when the repo is empty, and when the Github App doesn't have content access

## Goliac v0.15.7

- allow to use a PAT (Personal Access Token) to run Goliac (in particular useful to scaffold)

## Goliac v0.15.6

- new property to specify default branch for each repository

## Goliac v0.15.5

- ruleset bugfixes: include/exclude refs/heads prefix + required_status_checks payload

## Goliac v0.15.4

- introducing opentelemetry tracing

## Goliac v0.15.3

- allow public,internal,private repository visibility
- allow to restrict public repositories via global rules

## Goliac v0.15.2

- refactor error handling
- fix ruleset repository_ids API call
- dont enforce ruleset/branch protections on archived repositories
- adding support for ruleset parameters required_linear_history, non_fast_forward

## Goliac v0.15.1

- branch protections bugfixes

## Goliac v0.15.0

- introducing branch protections

## Goliac v0.14.1

- different security updates (golang/npm)
- bugfix: wrong api path for repository rulesets

## Goliac v0.14.0

- moving the project to goliac-project Github organization

## Goliac v0.13.3

- introducing ruleset rules `creation`, `update`, `deletion` and `non_fast_forward`

## Goliac v0.13.2

- bugfix on repository rulesets (name of the repository vs of the ruleset)

## Goliac v0.13.1

- bugfix on repository rulesets (bypass doesn't work on repository rulesets)
- do not touch organization rulesets when scanning repositories rulesets

## Goliac v0.13.0

- Introducing Repository Rulesets

## Goliac v0.12.0

- Introducing renaming of repository
