package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/goliac-project/goliac/internal"
	"github.com/goliac-project/goliac/internal/config"
	"github.com/goliac-project/goliac/internal/notification"
	"github.com/goliac-project/goliac/internal/observability"
	"github.com/schollz/progressbar/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var dryrunParameter bool
var forceParameter bool
var repositoryParameter string
var branchParameter string
var noProgressbar bool
var goliacAdminTeamnameParameter string
var usersOnly bool

type ProgressBar struct {
	bar *progressbar.ProgressBar
}

func CreateProgressBar() *ProgressBar {
	return &ProgressBar{bar: nil}
}

func (p *ProgressBar) Init(nbTotalAssets int) {
	bar := progressbar.NewOptions(nbTotalAssets,
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetDescription("fetching github"),
		//progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(36),
		progressbar.OptionShowTotalBytes(true),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		//progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
	p.bar = bar
}

func (p *ProgressBar) Extend(nbAssets int) {
	if p.bar == nil {
		return
	}
	p.bar.AddMax(nbAssets)
}

func (p *ProgressBar) LoadingAsset(entity string, nb int) {
	if p.bar == nil {
		return
	}
	p.bar.Add(nb)
}

func main() {
	verifyCmd := &cobra.Command{
		Use:   "verify <path>",
		Short: "Verify the validity of IAC directory structure",
		Long:  `Verify the validity of IAC directory structure`,
		Args:  cobra.MatchAll(cobra.MinimumNArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			path := args[0]
			goliac, err := internal.NewGoliacLightImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			logsCollector := observability.NewLogCollection()
			goliac.Validate(path, logsCollector)
			if logsCollector.HasErrors() {
				logrus.Errorf("failed to verify:")
				for _, err := range logsCollector.Errors {
					logrus.Errorf("- %s", err)
				}
				os.Exit(1)
			}
			if logsCollector.HasWarns() {
				logrus.Warnf("Warnings:")
				for _, err := range logsCollector.Warns {
					logrus.Warnf("- %s", err)
				}
			}
			for _, info := range logsCollector.Logs {
				logrus.WithFields(info.Fields).Logf(info.LogLevel, info.Format, info.Args...)
			}
		},
	}

	planCmd := &cobra.Command{
		Use:   "plan [--repository https_team_repository_url] [--branch branch]",
		Short: "Check the validity of IAC directory structure against a Github organization",
		Long: `Check the validity of IAC directory structure against a Github organization.
repository: a remote repository in the form https://github.com/...
repository can be passed by parameter or by defining GOLIAC_SERVER_GIT_REPOSITORY env variable
branch can be passed by parameter or by defining GOLIAC_SERVER_GIT_BRANCH env variable`,
		Run: func(cmd *cobra.Command, args []string) {
			repo := repositoryParameter
			branch := branchParameter

			if repo == "" {
				repo = config.Config.ServerGitRepository
			}
			if branch == "" {
				branch = config.Config.ServerGitBranch
			}
			if repo == "" || branch == "" {
				logrus.Fatalf("missing arguments. Try --help")
			}

			if config.Config.LogrusLevel == "debug" || config.Config.LogrusLevel == "info" {
				fmt.Println("Please wait, it can take several minutes to load everything. \u2615")
			}
			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			if !noProgressbar {
				bar := CreateProgressBar()
				err := goliac.SetRemoteObservability(bar)
				if err != nil {
					logrus.Warnf("failed to set remote observability: %s", err)
				}
			}

			ctx := context.Background()
			var span trace.Span
			if config.Config.OpenTelemetryEnabled {
				tracer := otel.Tracer("goliac")
				ctx, span = tracer.Start(ctx, "plan")
			}

			fs := osfs.New("/")

			logsCollector := observability.NewLogCollection()
			goliac.Apply(ctx, logsCollector, fs, true, repo, branch)
			if span != nil {
				span.End()
				config.ShutdownTraceProvider()
			}
			if logsCollector.HasErrors() {
				logrus.Errorf("Failed to plan:")
				for _, err := range logsCollector.Errors {
					logrus.Errorf("- %s", err)
				}
			}
			if logsCollector.HasWarns() {
				logrus.Warnf("Warnings:")
				for _, err := range logsCollector.Warns {
					logrus.Warnf("- %s", err)
				}
			}
			for _, info := range logsCollector.Logs {
				logrus.WithFields(info.Fields).Logf(info.LogLevel, info.Format, info.Args...)
			}
		},
	}

	planCmd.Flags().StringVarP(&repositoryParameter, "repository", "r", config.Config.ServerGitRepository, "repository (default env variable GOLIAC_SERVER_GIT_REPOSITORY)")
	planCmd.Flags().StringVarP(&branchParameter, "branch", "b", config.Config.ServerGitBranch, "branch (default env variable GOLIAC_SERVER_GIT_BRANCH)")
	planCmd.Flags().BoolVarP(&noProgressbar, "noprogressbar", "p", false, "display a progress bar")

	applyCmd := &cobra.Command{
		Use:   "apply [--repository https_team_repository_url] [--branch branch]",
		Short: "Verify and apply a IAC directory structure to a Github organization",
		Long: `Apply a IAC directory structure to a Github organization.
repository: a remote repository in the form https://github.com/...
repository can be passed by parameter or by defining GOLIAC_SERVER_GIT_REPOSITORY env variable
branch can be passed by parameter or by defining GOLIAC_SERVER_GIT_BRANCH env variable`,
		Run: func(cmd *cobra.Command, args []string) {
			repo := repositoryParameter
			branch := branchParameter

			if repo == "" {
				repo = config.Config.ServerGitRepository
			}
			if branch == "" {
				branch = config.Config.ServerGitBranch
			}
			if repo == "" || branch == "" {
				logrus.Fatalf("missing arguments, try --help")
			}

			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}

			if !noProgressbar {
				bar := CreateProgressBar()
				err := goliac.SetRemoteObservability(bar)
				if err != nil {
					logrus.Warnf("failed to set remote observability: %s", err)
				}
			}

			ctx := context.Background()
			var span trace.Span
			if config.Config.OpenTelemetryEnabled {
				tracer := otel.Tracer("goliac")
				ctx, span = tracer.Start(ctx, "apply")
			}

			fs := osfs.New("/")
			logsCollector := observability.NewLogCollection()
			goliac.Apply(ctx, logsCollector, fs, false, repo, branch)
			if span != nil {
				span.End()
				config.ShutdownTraceProvider()
			}
			if logsCollector.HasErrors() {
				logrus.Errorf("Failed to apply:")
				for _, err := range logsCollector.Errors {
					logrus.Errorf("- %s", err)
				}
			}
			if logsCollector.HasWarns() {
				logrus.Warnf("Warnings:")
				for _, err := range logsCollector.Warns {
					logrus.Warnf("- %s", err)
				}
			}
			for _, info := range logsCollector.Logs {
				logrus.WithFields(info.Fields).Logf(info.LogLevel, info.Format, info.Args...)
			}
		},
	}
	applyCmd.Flags().StringVarP(&repositoryParameter, "repository", "r", config.Config.ServerGitRepository, "repository (default env variable GOLIAC_SERVER_GIT_REPOSITORY)")
	applyCmd.Flags().StringVarP(&branchParameter, "branch", "b", config.Config.ServerGitBranch, "branch (default env variable GOLIAC_SERVER_GIT_BRANCH)")
	applyCmd.Flags().BoolVarP(&noProgressbar, "noprogressbar", "p", false, "display a progress bar")

	postSyncUsersCmd := &cobra.Command{
		Use:   "syncusers [--repository https_team_repository_url] [--branch branch] [--dryrun] [--force]",
		Short: "Update and commit users and teams definition",
		Long: `This command will use a user sync plugin to adjust users
 and team yaml definition, and commit them.
 repository: a remote repository in the form https://github.com/...
 branch: the branch to commit to.
 repository can be passed by parameter or by defining GOLIAC_SERVER_GIT_REPOSITORY env variable
 branch can be passed by parameter or by defining GOLIAC_SERVER_GIT_BRANCH env variable`,
		Args: cobra.MatchAll(cobra.MinimumNArgs(2), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			repo := repositoryParameter
			branch := branchParameter

			if repo == "" {
				repo = config.Config.ServerGitRepository
			}
			if branch == "" {
				branch = config.Config.ServerGitBranch
			}
			if repo == "" || branch == "" {
				logrus.Fatalf("missing arguments, try --help")
			}

			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			ctx := context.Background()
			var span trace.Span
			if config.Config.OpenTelemetryEnabled {
				tracer := otel.Tracer("goliac")
				ctx, span = tracer.Start(ctx, "syncusers")
			}

			fs := osfs.New("/")
			logsCollector := observability.NewLogCollection()
			goliac.UsersUpdate(ctx, logsCollector, fs, repo, branch, dryrunParameter, forceParameter)
			if span != nil {
				span.End()
				config.ShutdownTraceProvider()
			}
			if logsCollector.HasErrors() {
				logrus.Fatalf("failed to update and commit teams:")
				for _, err := range logsCollector.Errors {
					logrus.Errorf("- %s", err)
				}
			}
			for _, info := range logsCollector.Logs {
				logrus.WithFields(info.Fields).Logf(info.LogLevel, info.Format, info.Args...)
			}
		},
	}
	postSyncUsersCmd.Flags().StringVarP(&repositoryParameter, "repository", "r", config.Config.ServerGitRepository, "repository (default env variable GOLIAC_SERVER_GIT_REPOSITORY)")
	postSyncUsersCmd.Flags().StringVarP(&branchParameter, "branch", "b", config.Config.ServerGitBranch, "branch (default env variable GOLIAC_SERVER_GIT_BRANCH)")
	postSyncUsersCmd.Flags().BoolVarP(&dryrunParameter, "dryrun", "d", false, "dryrun mode")
	postSyncUsersCmd.Flags().BoolVarP(&forceParameter, "force", "f", false, "force mode")

	scaffoldcmd := &cobra.Command{
		Use:   "scaffold <directory> [--adminteam goliac_admin_team_name] [--users-only]",
		Short: "Will create a base directory based on your current Github organization",
		Long: `Base on your Github organization, this command will try to scaffold a
goliac directory to let you start with something.
The adminteam is your current team that contains Github administrator`,
		Args: cobra.MatchAll(cobra.MinimumNArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			directory := args[0]
			adminteam := goliacAdminTeamnameParameter
			if directory == "" || adminteam == "" {
				logrus.Fatalf("missing arguments. Try --help")
			}
			scaffold, err := internal.NewScaffold()
			if err != nil {
				logrus.Fatalf("failed to create scaffold: %s", err)
			}
			fmt.Println("Generating the IAC structure, it can take several minutes to list everything. \u2615")

			if !noProgressbar {
				bar := CreateProgressBar()
				err := scaffold.SetRemoteObservability(bar)
				if err != nil {
					logrus.Warnf("failed to set remote observability: %s", err)
				}
			}

			ctx := context.Background()
			var span trace.Span
			if config.Config.OpenTelemetryEnabled {
				tracer := otel.Tracer("goliac")
				ctx, span = tracer.Start(ctx, "scaffold")
			}

			logsCollector := observability.NewLogCollection()
			scaffold.Generate(ctx, directory, adminteam, usersOnly, logsCollector)
			if span != nil {
				span.End()
				config.ShutdownTraceProvider()
			}
			if logsCollector.HasErrors() {
				logrus.Fatalf("failed to create scaffold direcrory:")
				for _, err := range logsCollector.Errors {
					logrus.Errorf("- %s", err)
				}
			}
			if logsCollector.HasWarns() {
				logrus.Warnf("Warnings:")
				for _, err := range logsCollector.Warns {
					logrus.Warnf("- %s", err)
				}
			}
			for _, info := range logsCollector.Logs {
				logrus.WithFields(info.Fields).Logf(info.LogLevel, info.Format, info.Args...)
			}

			if !logsCollector.HasErrors() {
				newRepoSuggestion := filepath.Dir(directory)
				cwd, err := os.Getwd()
				if err == nil {
					newRepoSuggestion = filepath.Base(filepath.Join(cwd, directory))
				}
				newRepoSuggestion = "https://github.com/" + config.Config.GithubAppOrganization + "/" + newRepoSuggestion
				fmt.Printf(`Scaffold directory (%s) created
Now you can push this directory as a new repository to Github, like:
- check the validity of the directory:
   goliac verify %s
- create a new repository '%s' on Github
- push this directory to the new repository:
   cd %s
   git init --initial-branch=main
   git add .
   git commit -m 'team repository created'
   git remote add origin %s
   git push -u origin main
- check the validity of the repository:
   goliac plan --repository %s
- apply the repository:
   goliac apply --repository %s
- and then setup and start the goliac server
`, directory, directory, newRepoSuggestion, directory, newRepoSuggestion, newRepoSuggestion, newRepoSuggestion)
			}
		},
	}
	scaffoldcmd.Flags().StringVarP(&goliacAdminTeamnameParameter, "adminteam", "a", "goliac-admin", "name of the goliac admin team")
	scaffoldcmd.Flags().BoolVarP(&noProgressbar, "noprogressbar", "p", false, "display a progress bar")
	scaffoldcmd.Flags().BoolVarP(&usersOnly, "users-only", "u", false, "do not scaffold teams (except the admin) and repositories")

	servecmd := &cobra.Command{
		Use:   "serve",
		Short: "This will start the application in server mode",
		Long: `This will start the application in server mode, which will
apply periodically (env:GOLIAC_SERVER_APPLY_INTERVAL)
any changes from the teams Git repository to Github.`,
		Run: func(cmd *cobra.Command, args []string) {
			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			notificationService := notification.NewNullNotificationService()
			if config.Config.SlackToken != "" && config.Config.SlackChannel != "" {
				slackService := notification.NewSlackNotificationService(config.Config.SlackToken, config.Config.SlackChannel)
				notificationService = slackService
			}

			server := internal.NewGoliacServer(goliac, notificationService)
			server.Serve()
		},
	}

	versioncmd := &cobra.Command{
		Use:   "version",
		Short: "Return the version of the goliac CLI",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.GoliacBuildVersion)
		},
	}

	rootCmd := &cobra.Command{
		Use:   "goliac",
		Short: "A CLI for the goliac organization",
		Long: `a CLI library for goliac (GithHub Organization Sync Tool.
This CLI can mainly be plan (verify) or apply a IAC style directory structure to Github
Either local directory, or remote git repository`,
	}

	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(postSyncUsersCmd)
	rootCmd.AddCommand(scaffoldcmd)
	rootCmd.AddCommand(servecmd)
	rootCmd.AddCommand(versioncmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
