[//]: # (based on https://blog.pragmaticengineer.com/scaling-engineering-teams-via-writing-things-down-rfcs/)
# Management and visibility of the access of GitHub's teams to GitHub's repositories


## Abstract (what is the project about?)
[//]: # (write down all the details that will help one to understand the context around the discussion)
There is a gap between the team that manages the teams in GitHub and the users' reality.
The IT team is often responsible for configuring the teams, while the product developers are part of them.
The project works to allow the product developers to control their repositories.
A solution often seen is promoting some product developers as admins of their repositories.
Then the ownership of the repositories is shared between several teams with different agendas.

Our approach wants to reduce the number of repository admins to a strict minimum.
It increases the security of the platform.
If an account is compromised, its access is low.
It also prevents the spread of promotion.
Product developers with admin access promote someone else to the admin level.
This promotion is done without the context or the knowledge of the implications of such permissions.

### Provide a self-service solution for the company employees to manage teams and repositories on GitHub organization.

We start with the teams & users for security reasons.
When someone leaves an organization, we want to immediately remove the related Github user.

The solution enforces the usage of teams instead of users for the distribution of permissions


#### Out of scope: 
- Add or remove users to the Github's organization from an external source automatically.
- The management of protection rules will be part of a separated RFC.
- A migration plan will be part of a separated RFC.




[//]: # (2. Enforce company rules for security purposes and compliance in particular:)
[//]: # (   1. Audit and traceability of the activities on the github organization.)

## Architecture changes
* only teams, no users
 
  Internal users
  internal teams
  external users

## Service SLAs
- ??

## Service dependencies

- REST github api
- GraphQL github api

## Load & performance testing
- throttling github

## Github vs Github enterprise
- api version will be different.
- How many version behind do we support for github enterprise?

## Security considerations
- The github app cannot expose its credentials.
- One cannot promote himself as admin using the app

## Testing & roll-out
- Is there a solution to have a "local" github to test against for integration testing?
  - goVCR

## Metrics & monitoring
We will have infrastructure metrics and security metrics
OpenTelemetry

Where are the logs?
### Infra
- throttling
- which repository is managed vs no managed?

### Security


## Customer support considerations
