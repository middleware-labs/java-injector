package cli

// Re-export types from the types package to maintain clean API
import "github.com/middleware-labs/java-injector/pkg/cli/types"

type (
	CommandHandler   = types.CommandHandler
	CommandConfig    = types.CommandConfig
	NoArgsCommand    = types.NoArgsCommand
	SingleArgCommand = types.SingleArgCommand
)

// NewDefaultConfig returns default configuration values
func NewDefaultConfig() *CommandConfig {
	return types.NewDefaultConfig()
}
