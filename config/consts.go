package config

const (
	// DefaultMakeChannelCommand is the default command name for the create-temp-channel command.
	DefaultMakeChannelCommand = "mkch"

	// DefaultCommandPrefix is the default command prefix.
	DefaultCommandPrefix = "!"

	// ValidPrefixes is the list of all valid command prefixes.
	// It doesn't contain any character that could be used as markdown, as that may cause confusion for users when the command gets parsed as markdown.
	ValidPrefixes = "!@#$%^&=+()[]{};:'.,/?<>"
)
