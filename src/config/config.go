package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Environment represents the runtime environment
type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
	Test        Environment = "test"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// Config holds the application configuration
type Config struct {
	// Environment
	Environment Environment `mapstructure:"NODE_ENV"`
	Port        int         `mapstructure:"PORT"`

	// Logging
	LogLevel LogLevel `mapstructure:"LOG_LEVEL"`

	// Google Drive / rclone
	RcloneRemoteName string   `mapstructure:"RCLONE_REMOTE_NAME"`
	RcloneDirectories []string `mapstructure:"RCLONE_DIRECTORIES"`
	RcloneExcludes    []string `mapstructure:"RCLONE_EXCLUDES"`

	// Notion Integration
	NotionToken      string   `mapstructure:"NOTION_TOKEN"`
	NotionDatabaseID string   `mapstructure:"NOTION_DATABASE_ID"`
	NotionPageIDs    []string `mapstructure:"NOTION_PAGE_IDS"`

	// AI Providers
	OpenAIAPIKey string `mapstructure:"OPENAI_API_KEY"`
	GeminiAPIKey string `mapstructure:"GEMINI_API_KEY"`
	GrokAPIKey   string `mapstructure:"GROK_API_KEY"`

	// System Update
	SystemUpdateInterval string `mapstructure:"SYSTEM_UPDATE_INTERVAL"`
	SystemUpdateAuto     bool   `mapstructure:"SYSTEM_UPDATE_AUTO"`

	// Health Monitoring
	MonitorInterval string `mapstructure:"MONITOR_INTERVAL"`
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	v := viper.New()

	// Set config file
	v.SetConfigFile(".env")
	v.SetConfigType("env")

	// Set defaults
	setDefaults(v)

	// Read from .env file if it exists
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// .env file not found, continue with environment variables only
	}

	// Environment variables override .env file
	v.AutomaticEnv()

	// Parse configuration
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Parse comma-separated lists
	cfg.parseCommaSeparatedFields(v)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	v.SetDefault("NODE_ENV", "development")
	v.SetDefault("PORT", 3000)
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("RCLONE_REMOTE_NAME", "gdrive")
	v.SetDefault("SYSTEM_UPDATE_INTERVAL", "6h")
	v.SetDefault("SYSTEM_UPDATE_AUTO", false)
	v.SetDefault("MONITOR_INTERVAL", "60s")
}

// parseCommaSeparatedFields parses comma-separated string fields into slices
func (c *Config) parseCommaSeparatedFields(v *viper.Viper) {
	// Parse rclone directories
	if dirs := v.GetString("RCLONE_DIRECTORIES"); dirs != "" {
		c.RcloneDirectories = splitAndTrim(dirs)
	}

	// Parse rclone excludes
	if excludes := v.GetString("RCLONE_EXCLUDES"); excludes != "" {
		c.RcloneExcludes = splitAndTrim(excludes)
	}

	// Parse Notion page IDs
	if pageIDs := v.GetString("NOTION_PAGE_IDS"); pageIDs != "" {
		c.NotionPageIDs = splitAndTrim(pageIDs)
	}
}

// splitAndTrim splits a comma-separated string and trims whitespace
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate environment
	switch c.Environment {
	case Development, Production, Test:
		// Valid
	default:
		return fmt.Errorf("invalid environment: %s (must be development, production, or test)", c.Environment)
	}

	// Validate log level
	switch c.LogLevel {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		// Valid
	default:
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	// Validate port
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1 and 65535)", c.Port)
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == Development
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == Production
}

// IsTest returns true if running in test mode
func (c *Config) IsTest() bool {
	return c.Environment == Test
}

// GetRcloneDirectories returns the rclone directories or defaults
func (c *Config) GetRcloneDirectories() []string {
	if len(c.RcloneDirectories) > 0 {
		return c.RcloneDirectories
	}

	// Default directories
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}

	return []string{
		homeDir + "/Documents",
		homeDir + "/Pictures",
		homeDir + "/Videos",
		homeDir + "/Music",
		homeDir + "/.config",
		homeDir + "/Source",
	}
}

// GetRcloneExcludes returns the rclone exclude patterns or defaults
func (c *Config) GetRcloneExcludes() []string {
	if len(c.RcloneExcludes) > 0 {
		return c.RcloneExcludes
	}

	// Default exclusions (from TypeScript)
	return []string{
		// Build artifacts
		"**/node_modules/**",
		"**/.next/**",
		"**/dist/**",
		"**/build/**",
		"**/.nuxt/**",
		"**/.output/**",
		"**/.cache/**",
		"**/.parcel-cache/**",
		"**/.turbo/**",

		// Version control
		"**/.git/**",
		"**/.svn/**",
		"**/.hg/**",

		// Dependencies
		"**/vendor/**",
		"**/target/**",
		"**/__pycache__/**",
		"**/.venv/**",
		"**/venv/**",

		// IDE
		"**/.vscode/**",
		"**/.idea/**",
		"**/*.swp",
		"**/*.swo",

		// OS files
		"**/.DS_Store",
		"**/Thumbs.db",
		"**/desktop.ini",

		// Temporary files
		"**/*.tmp",
		"**/*.temp",
		"**/*.log",
		"**/*~",

		// Large media
		"**/*.iso",
		"**/*.img",
		"**/*.dmg",

		// Archives (already compressed)
		"**/*.zip",
		"**/*.tar",
		"**/*.tar.gz",
		"**/*.tgz",
		"**/*.rar",
		"**/*.7z",
	}
}

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf("Config{Environment=%s, Port=%d, LogLevel=%s, RcloneRemote=%s}",
		c.Environment, c.Port, c.LogLevel, c.RcloneRemoteName)
}
