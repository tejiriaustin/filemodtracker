package server

import (
	"errors"
	"regexp"
	"runtime"
	"strings"
)

func validateAndSanitizeCommand(cmd string) ([]string, error) {
	cmd = strings.TrimSpace(cmd)

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

	if baseCmd == "osqueryi" || baseCmd == "osqueryd" {
		if len(parts) < 2 {
			return nil, errors.New("invalid osquery command")
		}
		query := strings.Join(parts[1:], " ")
		if err := validateOsqueryQuery(query); err != nil {
			return nil, err
		}
		return parts, nil
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
	allowedCommands := map[string]bool{
		"ls": true, "cat": true, "grep": true, "echo": true,
		"ps": true, "top": true, "df": true, "du": true,
		"osqueryi": true, "osqueryd": true,
		"dir": true, "type": true, "findstr": true,
		"tasklist": true, "systeminfo": true, "chkdsk": true,
		"pwd": true,
	}

	return allowedCommands[cmd]
}

func validateOsqueryQuery(query string) error {
	query = strings.TrimSpace(strings.ToUpper(query))
	if !strings.HasPrefix(query, "SELECT") {
		return errors.New("only SELECT statements are allowed for osquery")
	}

	// Basic validation to prevent obvious SQL injection attempts
	dangerousKeywords := []string{"INSERT", "UPDATE", "DELETE", "DROP", "TRUNCATE", "ALTER", "--"}
	for _, keyword := range dangerousKeywords {
		if strings.Contains(query, keyword) {
			return errors.New("potentially dangerous osquery statement")
		}
	}

	return nil
}

func sanitizeArgument(arg string) (string, error) {
	reg, err := regexp.Compile(`[^a-zA-Z0-9_\-./\\]+`)
	if err != nil {
		return "", err
	}
	sanitized := reg.ReplaceAllString(arg, "")

	if strings.Contains(sanitized, "..") {
		return "", errors.New("invalid argument: potential path traversal")
	}

	osqueryArgs := map[string]bool{
		"--verbose":            true,
		"--json":               true,
		"--config_path":        true,
		"--database_path":      true,
		"--extension":          true,
		"--flagfile":           true,
		"--logger_path":        true,
		"--pidfile":            true,
		"--disable_extensions": true,
		"--disable_database":   true,
		"--disable_events":     true,
		"--disable_audit":      true,
		"--disable_watchdog":   true,
		"--enable_monitor":     true,
		"--force":              true,
		"--allow_unsafe":       true,
	}

	if strings.HasPrefix(sanitized, "--") && !osqueryArgs[sanitized] {
		return "", errors.New("invalid osquery argument")
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
