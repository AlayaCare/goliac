package internal

import (
	"fmt"

	"github.com/Alayacare/goliac/internal/config"
	"github.com/Alayacare/goliac/internal/engine"
)

/**
 * Each command/mutation we want to perform will be isloated into a GithubCommand
 * object, so we can regroup all of them to apply (or cancel) them in batch
 */
type GithubCommand interface {
	Apply()
}

/*
 * GithubBatchExecutor will collects all commands to apply
 * if there the number of changes to apply is not too big, it will apply on the `Commit()`
 * Usage:
 * gal := NewGithubBatchExecutor(client)
 * gal.Begin()
 * gal.Create...
 * gal.Update...
 * ...
 * gal.Commit()
 */
type GithubBatchExecutor struct {
	client        engine.ReconciliatorExecutor
	maxChangesets int
	commands      []GithubCommand
}

func NewGithubBatchExecutor(client engine.ReconciliatorExecutor, maxChangesets int) *GithubBatchExecutor {
	gal := GithubBatchExecutor{
		client:        client,
		maxChangesets: maxChangesets,
		commands:      make([]GithubCommand, 0),
	}
	return &gal
}

func (g *GithubBatchExecutor) AddUserToOrg(dryrun bool, ghuserid string) {
	g.commands = append(g.commands, &GithubCommandAddUserToOrg{
		client:   g.client,
		dryrun:   dryrun,
		ghuserid: ghuserid,
	})
}

func (g *GithubBatchExecutor) RemoveUserFromOrg(dryrun bool, ghuserid string) {
	g.commands = append(g.commands, &GithubCommandAddUserToOrg{
		client:   g.client,
		dryrun:   dryrun,
		ghuserid: ghuserid,
	})
}

func (g *GithubBatchExecutor) CreateTeam(dryrun bool, teamname string, description string, members []string) {
	g.commands = append(g.commands, &GithubCommandCreateTeam{
		client:      g.client,
		dryrun:      dryrun,
		teamname:    teamname,
		description: description,
		members:     members,
	})
}

// role = member or maintainer (usually we use member)
func (g *GithubBatchExecutor) UpdateTeamAddMember(dryrun bool, teamslug string, username string, role string) {
	g.commands = append(g.commands, &GithubCommandUpdateTeamAddMember{
		client:   g.client,
		dryrun:   dryrun,
		teamslug: teamslug,
		member:   username,
		role:     role,
	})
}

func (g *GithubBatchExecutor) UpdateTeamRemoveMember(dryrun bool, teamslug string, username string) {
	g.commands = append(g.commands, &GithubCommandUpdateTeamRemoveMember{
		client:   g.client,
		dryrun:   dryrun,
		teamslug: teamslug,
		member:   username,
	})
}

func (g *GithubBatchExecutor) DeleteTeam(dryrun bool, teamslug string) {
	g.commands = append(g.commands, &GithubCommandDeleteTeam{
		client:   g.client,
		dryrun:   dryrun,
		teamslug: teamslug,
	})
}

func (g *GithubBatchExecutor) CreateRepository(dryrun bool, reponame string, description string, writers []string, readers []string, boolProperties map[string]bool) {
	g.commands = append(g.commands, &GithubCommandCreateRepository{
		client:         g.client,
		dryrun:         dryrun,
		reponame:       reponame,
		description:    description,
		readers:        readers,
		writers:        writers,
		boolProperties: boolProperties,
	})
}

func (g *GithubBatchExecutor) UpdateRepositoryAddTeamAccess(dryrun bool, reponame string, teamslug string, permission string) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositoryAddTeamAccess{
		client:     g.client,
		dryrun:     dryrun,
		reponame:   reponame,
		teamslug:   teamslug,
		permission: permission,
	})
}

func (g *GithubBatchExecutor) UpdateRepositoryUpdateTeamAccess(dryrun bool, reponame string, teamslug string, permission string) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositoryUpdateTeamAccess{
		client:     g.client,
		dryrun:     dryrun,
		reponame:   reponame,
		teamslug:   teamslug,
		permission: permission,
	})
}

func (g *GithubBatchExecutor) UpdateRepositoryRemoveTeamAccess(dryrun bool, reponame string, teamslug string) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositoryRemoveTeamAccess{
		client:   g.client,
		dryrun:   dryrun,
		reponame: reponame,
		teamslug: teamslug,
	})
}

func (g *GithubBatchExecutor) UpdateRepositoryUpdateBoolProperty(dryrun bool, reponame string, propertyName string, propertyValue bool) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositoryUpdateBoolProperty{
		client:        g.client,
		dryrun:        dryrun,
		reponame:      reponame,
		propertyName:  propertyName,
		propertyValue: propertyValue,
	})
}

func (g *GithubBatchExecutor) UpdateRepositorySetExternalUser(dryrun bool, reponame string, githubid string, permission string) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositorySetExternalUser{
		client:     g.client,
		dryrun:     dryrun,
		reponame:   reponame,
		githubid:   githubid,
		permission: permission,
	})
}

func (g *GithubBatchExecutor) UpdateRepositoryRemoveExternalUser(dryrun bool, reponame string, githubid string) {
	g.commands = append(g.commands, &GithubCommandUpdateRepositoryRemoveExternalUser{
		client:   g.client,
		dryrun:   dryrun,
		reponame: reponame,
		githubid: githubid,
	})
}

func (g *GithubBatchExecutor) DeleteRepository(dryrun bool, reponame string) {
	g.commands = append(g.commands, &GithubCommandDeleteRepository{
		client:   g.client,
		dryrun:   dryrun,
		reponame: reponame,
	})
}

func (g *GithubBatchExecutor) AddRuleset(dryrun bool, ruleset *engine.GithubRuleSet) {
	g.commands = append(g.commands, &GithubCommandAddRuletset{
		client:  g.client,
		dryrun:  dryrun,
		ruleset: ruleset,
	})
}

func (g *GithubBatchExecutor) UpdateRuleset(dryrun bool, ruleset *engine.GithubRuleSet) {
	g.commands = append(g.commands, &GithubCommandUpdateRuletset{
		client:  g.client,
		dryrun:  dryrun,
		ruleset: ruleset,
	})
}

func (g *GithubBatchExecutor) DeleteRuleset(dryrun bool, rulesetid int) {
	g.commands = append(g.commands, &GithubCommandDeleteRuletset{
		client:    g.client,
		dryrun:    dryrun,
		rulesetid: rulesetid,
	})
}

func (g *GithubBatchExecutor) Begin(dryrun bool) {
	g.commands = make([]GithubCommand, 0)
}
func (g *GithubBatchExecutor) Rollback(dryrun bool, err error) {
	g.commands = make([]GithubCommand, 0)
}
func (g *GithubBatchExecutor) Commit(dryrun bool) error {
	if len(g.commands) > g.maxChangesets && !config.Config.MaxChangesetsOverride {
		return fmt.Errorf("more than %d changesets to apply (total of %d), this is suspicious. Aborting (see Goliac troubleshooting guide for help)", g.maxChangesets, len(g.commands))
	}
	for _, c := range g.commands {
		c.Apply()
	}
	g.commands = make([]GithubCommand, 0)
	return nil
}

type GithubCommandAddUserToOrg struct {
	client   engine.ReconciliatorExecutor
	dryrun   bool
	ghuserid string
}

func (g *GithubCommandAddUserToOrg) Apply() {
	g.client.AddUserToOrg(g.dryrun, g.ghuserid)
}

type GithubCommandCreateRepository struct {
	client         engine.ReconciliatorExecutor
	dryrun         bool
	reponame       string
	description    string
	writers        []string
	readers        []string
	boolProperties map[string]bool
}

func (g *GithubCommandCreateRepository) Apply() {
	g.client.CreateRepository(g.dryrun, g.reponame, g.description, g.writers, g.readers, g.boolProperties)
}

type GithubCommandCreateTeam struct {
	client      engine.ReconciliatorExecutor
	dryrun      bool
	teamname    string
	description string
	members     []string
}

func (g *GithubCommandCreateTeam) Apply() {
	g.client.CreateTeam(g.dryrun, g.teamname, g.description, g.members)
}

type GithubCommandDeleteRepository struct {
	client   engine.ReconciliatorExecutor
	dryrun   bool
	reponame string
}

func (g *GithubCommandDeleteRepository) Apply() {
	g.client.DeleteRepository(g.dryrun, g.reponame)
}

type GithubCommandDeleteTeam struct {
	client   engine.ReconciliatorExecutor
	dryrun   bool
	teamslug string
}

func (g *GithubCommandDeleteTeam) Apply() {
	g.client.DeleteTeam(g.dryrun, g.teamslug)
}

type GithubCommandRemoveUserFromOrg struct {
	client   engine.ReconciliatorExecutor
	dryrun   bool
	ghuserid string
}

func (g *GithubCommandRemoveUserFromOrg) Apply() {
	g.client.RemoveUserFromOrg(g.dryrun, g.ghuserid)
}

type GithubCommandUpdateRepositoryRemoveTeamAccess struct {
	client   engine.ReconciliatorExecutor
	dryrun   bool
	reponame string
	teamslug string
}

func (g *GithubCommandUpdateRepositoryRemoveTeamAccess) Apply() {
	g.client.UpdateRepositoryRemoveTeamAccess(g.dryrun, g.reponame, g.teamslug)
}

type GithubCommandUpdateRepositoryAddTeamAccess struct {
	client     engine.ReconciliatorExecutor
	dryrun     bool
	reponame   string
	teamslug   string
	permission string
}

func (g *GithubCommandUpdateRepositoryAddTeamAccess) Apply() {
	g.client.UpdateRepositoryAddTeamAccess(g.dryrun, g.reponame, g.teamslug, g.permission)
}

type GithubCommandUpdateRepositoryUpdateTeamAccess struct {
	client     engine.ReconciliatorExecutor
	dryrun     bool
	reponame   string
	teamslug   string
	permission string
}

func (g *GithubCommandUpdateRepositoryUpdateTeamAccess) Apply() {
	g.client.UpdateRepositoryUpdateTeamAccess(g.dryrun, g.reponame, g.teamslug, g.permission)
}

type GithubCommandUpdateRepositorySetExternalUser struct {
	client     engine.ReconciliatorExecutor
	dryrun     bool
	reponame   string
	githubid   string
	permission string
}

func (g *GithubCommandUpdateRepositorySetExternalUser) Apply() {
	g.client.UpdateRepositorySetExternalUser(g.dryrun, g.reponame, g.githubid, g.permission)
}

type GithubCommandUpdateRepositoryRemoveExternalUser struct {
	client   engine.ReconciliatorExecutor
	dryrun   bool
	reponame string
	githubid string
}

func (g *GithubCommandUpdateRepositoryRemoveExternalUser) Apply() {
	g.client.UpdateRepositoryRemoveExternalUser(g.dryrun, g.reponame, g.githubid)
}

type GithubCommandUpdateRepositoryUpdateBoolProperty struct {
	client        engine.ReconciliatorExecutor
	dryrun        bool
	reponame      string
	propertyName  string
	propertyValue bool
}

func (g *GithubCommandUpdateRepositoryUpdateBoolProperty) Apply() {
	g.client.UpdateRepositoryUpdateBoolProperty(g.dryrun, g.reponame, g.propertyName, g.propertyValue)
}

type GithubCommandUpdateTeamAddMember struct {
	client   engine.ReconciliatorExecutor
	dryrun   bool
	teamslug string
	member   string
	role     string
}

func (g *GithubCommandUpdateTeamAddMember) Apply() {
	g.client.UpdateTeamAddMember(g.dryrun, g.teamslug, g.member, g.role)
}

type GithubCommandUpdateTeamRemoveMember struct {
	client   engine.ReconciliatorExecutor
	dryrun   bool
	teamslug string
	member   string
}

func (g *GithubCommandUpdateTeamRemoveMember) Apply() {
	g.client.UpdateTeamRemoveMember(g.dryrun, g.teamslug, g.member)
}

type GithubCommandAddRuletset struct {
	client  engine.ReconciliatorExecutor
	dryrun  bool
	ruleset *engine.GithubRuleSet
}

func (g *GithubCommandAddRuletset) Apply() {
	g.client.AddRuleset(g.dryrun, g.ruleset)
}

type GithubCommandUpdateRuletset struct {
	client  engine.ReconciliatorExecutor
	dryrun  bool
	ruleset *engine.GithubRuleSet
}

func (g *GithubCommandUpdateRuletset) Apply() {
	g.client.UpdateRuleset(g.dryrun, g.ruleset)
}

type GithubCommandDeleteRuletset struct {
	client    engine.ReconciliatorExecutor
	dryrun    bool
	rulesetid int
}

func (g *GithubCommandDeleteRuletset) Apply() {
	g.client.DeleteRuleset(g.dryrun, g.rulesetid)
}
