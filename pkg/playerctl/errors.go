package playerctl

import (
	"errors"
	"fmt"
)

var (
	// ErrPlayerNotFound indicates that no player matching the selector could be found.
	ErrPlayerNotFound = errors.New("player not found")
)

// InvalidCommandError indicates that a command is not supported by the current player or context.
type InvalidCommandError struct {
	Command string
}

// Error implements the error interface.
func (e InvalidCommandError) Error() string {
	if e.Command == "" {
		return "invalid command"
	}
	return fmt.Sprintf("invalid command: %s", e.Command)
}

// FormatError reports a formatter parse or rendering error.
type FormatError struct {
	Message string
}

// Error implements the error interface.
func (e FormatError) Error() string {
	if e.Message == "" {
		return "format error"
	}
	return fmt.Sprintf("format error: %s", e.Message)
}
