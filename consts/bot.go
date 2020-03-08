package consts

import "regexp"

const (
	// DefaultMakeChannelCommand is the default command name for the create-temp-channel command.
	DefaultMakeChannelCommand = "mkch"

	// DefaultCommandPrefix is the default command prefix.
	DefaultCommandPrefix = "!"

	// ValidPrefixes is the list of all valid command prefixes.
	// It doesn't contain any character that could be used as markdown, as that may cause confusion for users when the command gets parsed as markdown.
	ValidPrefixes = "!@#$%^&=+()[]{};:'.,/?<>"

	// ValidCommandLettersDescription is the printable description of valid command letters.
	ValidCommandLettersDescription = "letters, underscores (_), and dashes (-)"
	// MinCommandNameLength is the minimum amount of required letters
	MinCommandNameLength = 2
	// MaxCommandNameLength is the maximum amount of allowed letters.
	MaxCommandNameLength = 32
)

var (
	// ValidCommandLettersRegex is the regexp of valid command letters.
	ValidCommandLettersRegex = regexp.MustCompile("^[A-Za-z-_]{2,32}$")
)
