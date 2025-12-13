package main

import (
	"os"

	daemira "github.com/ln64-git/daemira/internal"
	"github.com/ln64-git/daemira/src/cli"
	"github.com/ln64-git/daemira/src/config"
	"github.com/ln64-git/daemira/src/utility"
)

var (
	version = "0.1.0"
	logger  *utility.Logger
	daemon  *daemira.Daemira
)

func main() {
	// Check if running as root
	if os.Geteuid() == 0 {
		logger = utility.NewLogger("cli", utility.INFO)
		logger.Info("Running with root privileges")
	} else {
		logger = utility.NewLogger("cli", utility.INFO)
		logger.Info("Running as user (system updates will require sudo)")
	}
	defer logger.Close()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		logger.Warn("Failed to load config: %v, using defaults", err)
		cfg = &config.Config{
			RcloneRemoteName: "gdrive",
		}
	}

	// Initialize daemon
	daemon = daemira.NewDaemira(logger, cfg)

	// Create CLI and commands
	cliInstance := cli.NewCLI(daemon, logger)
	rootCmd := cliInstance.CreateCommands()

	// Execute
	if err := rootCmd.Execute(); err != nil {
		logger.Error("Error: %v", err)
		os.Exit(1)
	}
}
