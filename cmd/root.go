package cmd

import (
	"fmt"
	"os"

	"github.com/adhaniscuber/reprac/internal/config"
	"github.com/adhaniscuber/reprac/internal/github"
	"github.com/adhaniscuber/reprac/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var cfgPath string

var rootCmd = &cobra.Command{
	Use:   "reprac",
	Short: "TUI dashboard to track unreleased GitHub repo changes",
	Long: `reprac — never forget to deploy again.

Tracks your GitHub repositories and shows which ones have commits
on the default branch that haven't been released/tagged yet.

Examples:
  reprac                          # run with default config
  reprac --config ~/repos.yaml   # run with custom config
  reprac init                     # create a sample config file
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}

		gh := github.New()

		m := ui.New(cfgPath, cfg, gh)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err = p.Run()
		return err
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a sample repos.yaml config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(cfgPath); err == nil {
			fmt.Printf("Config already exists at %s\n", cfgPath)
			fmt.Print("Overwrite? [y/N] ")
			var ans string
			fmt.Scanln(&ans)
			if ans != "y" && ans != "Y" {
				fmt.Println("Aborted.")
				return nil
			}
		}
		if err := config.InitExample(cfgPath); err != nil {
			return fmt.Errorf("creating config: %w", err)
		}
		fmt.Printf("✅ Created sample config at: %s\n", cfgPath)
		fmt.Printf("   Edit it, then run: reprac\n")
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("reprac v0.1.2")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	defaultCfg := config.DefaultPath()
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", defaultCfg, "path to repos.yaml config file")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(versionCmd)
}
