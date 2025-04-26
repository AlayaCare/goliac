package entity

import (
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/goliac-project/goliac/internal/utils"
	"github.com/stretchr/testify/assert"
)

func fixtureCreateRuleSet(t *testing.T, fs billy.Filesystem) {
	fs.MkdirAll("rulesets", 0755)
	err := utils.WriteFile(fs, "rulesets/ruleset1.yaml", []byte(`
apiVersion: v1
kind: Ruleset
name: ruleset1
spec:
  ruleset:
    enforcement: evaluate
    bypassapps:
      - appname: goliac-project-app
        mode: always
    conditions:
      include: 
        - "~DEFAULT_BRANCH"

    rules:
      - ruletype: pull_request
        parameters:
          requiredApprovingReviewCount: 1
`), 0644)
	assert.Nil(t, err)

	err = utils.WriteFile(fs, "rulesets/ruleset2.yaml", []byte(`
apiVersion: v1
kind: Ruleset
name: ruleset2
spec:
  ruleset:
    enforcement: evaluate
    bypassapps:
      - appname: goliac-project-app
        mode: always
    conditions:
      include: 
        - "~DEFAULT_BRANCH"

    rules:
      - ruletype: required_status_checks
        parameters:
          requiredStatusChecks:
            - circleCI check
            - jenkins check
`), 0644)
	assert.Nil(t, err)
}

func TestRuleset(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateRuleSet(t, fs)

		errorCollector := observability.NewErrorCollection()
		rulesets := ReadRuleSetDirectory(fs, "rulesets", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, rulesets)
		assert.Equal(t, 2, len(rulesets))

	})
}

func TestRulesetParametersComparison(t *testing.T) {

	// happy path
	t.Run("happy path", func(t *testing.T) {
		// create a new user
		fs := memfs.New()
		fixtureCreateRuleSet(t, fs)

		errorCollector := observability.NewErrorCollection()
		rulesets := ReadRuleSetDirectory(fs, "rulesets", errorCollector)
		assert.Equal(t, false, errorCollector.HasErrors())
		assert.Equal(t, false, errorCollector.HasWarns())
		assert.NotNil(t, rulesets)

		res := CompareRulesetParameters(rulesets["ruleset1"].Spec.Ruleset.Rules[0].Ruletype, rulesets["ruleset1"].Spec.Ruleset.Rules[0].Parameters, rulesets["ruleset2"].Spec.Ruleset.Rules[0].Parameters)
		assert.False(t, res)

		res = CompareRulesetParameters(rulesets["ruleset1"].Spec.Ruleset.Rules[0].Ruletype, rulesets["ruleset1"].Spec.Ruleset.Rules[0].Parameters, rulesets["ruleset1"].Spec.Ruleset.Rules[0].Parameters)
		assert.True(t, res)

		res = CompareRulesetParameters(rulesets["ruleset2"].Spec.Ruleset.Rules[0].Ruletype, rulesets["ruleset2"].Spec.Ruleset.Rules[0].Parameters, rulesets["ruleset2"].Spec.Ruleset.Rules[0].Parameters)
		assert.True(t, res)
	})
}
