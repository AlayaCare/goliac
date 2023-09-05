package main

import (
	"fmt"
	"os"

	"github.com/Alayacare/goliac/internal"
	"github.com/Alayacare/goliac/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	verifyCmd := &cobra.Command{
		Use:   "verify [path]",
		Short: "Verify the validity of IAC directory structure",
		Long:  `Verify the validity of IAC directory structure`,
		Args:  cobra.MatchAll(cobra.MinimumNArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			path := args[0]
			goliac, err := internal.NewGoliacLightImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			err = goliac.Validate(path)
			if err != nil {
				logrus.Fatalf("failed to verify: %s", err)
			}
		},
	}

	planCmd := &cobra.Command{
		Use:   "plan [repository] [branch]",
		Short: "Check the validity of IAC directory structure against a Github organization",
		Long: `Check the validity of IAC directory structure against a Github organization.
repository: local or remote repository. A remote repository is in the form
https://github.com/...`,
		Args: cobra.MatchAll(cobra.MinimumNArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			repo := ""
			branch := ""

			if len(args) == 2 {
				repo = args[0]
				branch = args[1]
			} else {
				repo = config.Config.ServerGitRepository
				branch = config.Config.ServerGitBranch
			}
			if repo == "" || branch == "" {
				logrus.Fatalf("missing arguments")
			}

			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			err = goliac.Apply(true, repo, branch, true)
			if err != nil {
				logrus.Errorf("Failed to plan: %v", err)
			}
		},
	}

	applyCmd := &cobra.Command{
		Use:   "apply [repository] [branch]",
		Short: "Verify and apply a IAC directory structure to a Github organization",
		Long: `Apply a IAC directory structure to a Github organization.
repository: local or remote repository. A remote repository is in the form
https://github.com/...`,
		Run: func(cmd *cobra.Command, args []string) {
			repo := ""
			branch := ""

			if len(args) == 2 {
				repo = args[0]
				branch = args[1]
			} else {
				repo = config.Config.ServerGitRepository
				branch = config.Config.ServerGitBranch
			}
			if repo == "" || branch == "" {
				logrus.Fatalf("missing arguments")
			}

			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			err = goliac.Apply(false, repo, branch, true)
			if err != nil {
				logrus.Errorf("Failed to apply: %v", err)
			}
		},
	}

	postSyncUsersCmd := &cobra.Command{
		Use:   "syncusers [repository] [branch]",
		Short: "Update and commit users and teams definition",
		Long:  `This command will use a user sync plugin to adjust users and team yaml definition, and commit them`,
		Run: func(cmd *cobra.Command, args []string) {
			repo := ""
			branch := ""

			if len(args) == 2 {
				repo = args[0]
				branch = args[1]
			} else {
				repo = config.Config.ServerGitRepository
				branch = config.Config.ServerGitBranch
			}
			if repo == "" || branch == "" {
				logrus.Fatalf("missing arguments")
			}

			goliac, err := internal.NewGoliacImpl()
			if err != nil {
				logrus.Fatalf("failed to create goliac: %s", err)
			}
			err = goliac.UsersUpdate(repo, branch)
			if err != nil {
				logrus.Fatalf("failed to update and commit teams: %s", err)
			}
		},
	}

	scaffoldcmd := &cobra.Command{
		Use:   "scaffold [directory] [adminteam]",
		Short: "Will create a base directory based on your current Github organization",
		Long: `Base on your Github organization, this command will try to scaffold a
goliac directory to let you start with something.
The adminteam is your current team that contains Github administrator`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				logrus.Fatalf("missing arguments")
			}
			directory := args[0]
			adminteam := args[1]
			scaffold, err := internal.NewScaffold()
			if err != nil {
				logrus.Fatalf("failed to create scaffold: %s", err)
			}
			err = scaffold.Generate(directory, adminteam)
			if err != nil {
				logrus.Fatalf("failed to create scaffold direcrory: %s", err)
			}
		},
	}

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
			server := internal.NewGoliacServer(goliac)
			server.Serve()
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

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
