package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/adhaniscuber/reprac/internal/config"
	"github.com/adhaniscuber/reprac/internal/github"
	"github.com/adhaniscuber/reprac/internal/ui"
	"github.com/adhaniscuber/reprac/internal/web"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	cfgPath   string
	webNoOpen bool
	webPort   int
)

// Version is set at build time via -ldflags.
var Version = "dev"

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

		m := ui.New(cfgPath, cfg, gh, Version)
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

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Run the web UI (shadcn-styled) instead of the TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		gh := github.New()
		srv := web.NewServer(cfgPath, cfg, gh, Version)

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		url, wait, _, err := srv.Start(ctx, webPort)
		if err != nil {
			return fmt.Errorf("starting server: %w", err)
		}
		fmt.Printf("✨ reprac web UI: %s\n", url)
		fmt.Println("   press Ctrl+C to stop")
		if !webNoOpen {
			_ = openBrowserURL(url)
		}
		return wait()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("reprac " + Version)
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
	webCmd.Flags().BoolVar(&webNoOpen, "no-open", false, "do not auto-open the browser")
	webCmd.Flags().IntVarP(&webPort, "port", "p", web.DefaultPort, "port to bind (0 = random; falls back to random if busy)")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(webCmd)
}

func openBrowserURL(url string) error {
	var name string
	switch runtime.GOOS {
	case "darwin":
		name = "open"
	case "windows":
		name = "start"
	default:
		name = "xdg-open"
	}
	return exec.Command(name, url).Start()
}
