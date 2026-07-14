package agents

import (
	"context"

	"github.com/gentleman-programming/gentle-ai/internal/model"
	"github.com/gentleman-programming/gentle-ai/internal/system"
)

// Capability tags for adapter feature checks.
type Capability string

const (
	CapabilityAutoInstall Capability = "auto-install"
)

// Adapter is the core abstraction for AI agent integration. Components use
// adapter methods instead of switch statements on AgentID, making it trivial
// to add new agents without modifying component code.
type Adapter interface {
	// Identity
	Agent() model.AgentID
	Tier() model.SupportTier

	// Detection
	Detect(ctx context.Context, homeDir string) (installed bool, binaryPath string, configPath string, configFound bool, err error)

	// Installation
	SupportsAutoInstall() bool
	InstallCommand(profile system.PlatformProfile) ([][]string, error)

	// Config paths — components use these instead of hardcoding paths per agent.
	GlobalConfigDir(homeDir string) string
	SystemPromptDir(homeDir string) string
	SystemPromptFile(homeDir string) string
	SkillsDir(homeDir string) string
	SettingsPath(homeDir string) string

	// Config strategies — HOW to inject content, not WHERE (that's paths above).
	SystemPromptStrategy() model.SystemPromptStrategy
	MCPStrategy() model.MCPStrategy

	// MCP path resolution
	MCPConfigPath(homeDir string, serverName string) string

	// Optional capabilities — agents declare what they support.
	SupportsOutputStyles() bool
	OutputStyleDir(homeDir string) string

	SupportsSlashCommands() bool
	CommandsDir(homeDir string) string

	SupportsSubAgents() bool
	SubAgentsDir(homeDir string) string
	EmbeddedSubAgentsDir() string

	SupportsSkills() bool
	SupportsSystemPrompt() bool
	SupportsMCP() bool
}

// EffectiveCodeGraphWiringDetector is an optional adapter capability for agents
// whose configuration format requires semantic validation beyond marker checks.
type EffectiveCodeGraphWiringDetector interface {
	EffectiveCodeGraphWiring(homeDir string) (path string, configured bool)
}

// ThemeInjectionController is an optional adapter capability. Agents whose
// settings schema rejects a top-level "theme" key implement it and return
// false to opt out of theme injection into their settings file. Adapters that
// do not implement it are treated as supporting theme injection.
type ThemeInjectionController interface {
	SupportsThemeInjection() bool
}

// SupportsThemeInjection reports whether theme injection into the adapter's
// settings file is permitted. An adapter opts out by implementing
// ThemeInjectionController and returning false; adapters that do not implement
// it support theme injection by default. This is the single source of truth for
// the opt-out so that every consumer (injection and post-apply verification)
// agrees on which adapters receive a theme.
func SupportsThemeInjection(adapter Adapter) bool {
	controller, ok := adapter.(ThemeInjectionController)
	return !ok || controller.SupportsThemeInjection()
}
