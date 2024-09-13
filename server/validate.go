package server

import (
	"errors"
	"regexp"
	"runtime"
	"strings"
)

func validateAndSanitizeCommand(cmd string) ([]string, error) {
	cmd = strings.TrimSpace(cmd)

	// Basic structure check
	if len(cmd) == 0 {
		return nil, errors.New("empty command")
	}

	var parts []string
	if runtime.GOOS == "windows" {
		parts = splitWindowsCommand(cmd)
	} else {
		parts = strings.Fields(cmd)
	}

	baseCmd := strings.ToLower(parts[0])
	if !isAllowedCommand(baseCmd) {
		return nil, errors.New("base command not allowed")
	}

	for i := 1; i < len(parts); i++ {
		sanitized, err := sanitizeArgument(parts[i])
		if err != nil {
			return nil, err
		}
		parts[i] = sanitized
	}

	return parts, nil
}

// Performing a whitelist
// Another approach would be to blacklist some commands
func isAllowedCommand(cmd string) bool {
	unixCommands := map[string]bool{
		"ls": true, "cat": true, "grep": true, "echo": true,
		"ps": true, "top": true, "df": true, "du": true,
		// May need t0 add more Linux commands here
	}
	windowsCommands := map[string]bool{
		"dir": true, "type": true, "findstr": true, "echo": true,
		"tasklist": true, "systeminfo": true, "chkdsk": true,
		// May need to add more Windows commands here
	}

	if runtime.GOOS == "windows" {
		return windowsCommands[cmd]
	}
	return unixCommands[cmd]
}

func sanitizeArgument(arg string) (string, error) {
	// Remove any characters that aren't alphanumeric, underscore, hyphen, period, or forward slash
	reg, err := regexp.Compile(`[^a-zA-Z0-9_\-./\\]+`)
	if err != nil {
		return "", err
	}
	sanitized := reg.ReplaceAllString(arg, "")

	// Prevent path traversal attempts
	if strings.Contains(sanitized, "..") {
		return "", errors.New("invalid argument: potential path traversal")
	}

	return sanitized, nil
}
func splitWindowsCommand(cmd string) []string {
	var parts []string
	var current string
	inQuotes := false

	for _, r := range cmd {
		switch r {
		case '"':
			if inQuotes {
				current += string(r)
				inQuotes = false
			} else {
				current += string(r)
				inQuotes = true
			}
		case ' ':
			if inQuotes {
				current += string(r)
			} else {
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
			}
		default:
			current += string(r)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
