package server

import (
	"runtime"
	"testing"
)

func TestValidateAndSanitizeCommand(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
		expectError    bool
		onlyOS         string
	}{
		{"Empty command", "", "", true, ""},
		{"Simple valid command", "ls", "ls", false, "unix"},
		{"Valid command with args", "ls -l /home", "ls -l /home", false, "unix"},
		{"Valid Windows command", "dir C:\\Users", "dir C:\\Users", false, "windows"},
		{"Command with spaces", "echo Hello World", "echo Hello World", false, ""},
		{"Command with quotes", `echo "Hello World"`, `echo Hello World`, false, ""},
		{"Sanitize special chars", "ls file@with#special&chars.txt", "ls filewithspecialchars.txt", false, "unix"},
		{"Path traversal attempt", "cat ../../../etc/passwd", "", true, "unix"},
		{"Disallowed command", "rm -rf /", "", true, "unix"},
		{"Windows disallowed command", "del C:\\Windows\\System32", "", true, "windows"},
		{"Command with multiple spaces", "ps   aux", "ps aux", false, "unix"},
		{"Complex Windows command", `dir "C:\Program Files" /s`, `dir "C:\Program Files" /s`, false, "windows"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.onlyOS != "" && tt.onlyOS != runtime.GOOS {
				t.Skip("Skipping test for different OS")
			}

			output, err := validateAndSanitizeCommand(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected an error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if output != tt.expectedOutput {
				t.Errorf("Expected output %q, but got %q", tt.expectedOutput, output)
			}
		})
	}
}

func TestSanitizeArgument(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput string
		expectError    bool
	}{
		{"Simple argument", "file.txt", "file.txt", false},
		{"Argument with special chars", "file@#$%^&.txt", "file.txt", false},
		{"Path traversal attempt", "../../../etc/passwd", "", true},
		{"Argument with allowed special chars", "file-name_1.2.3", "file-name_1.2.3", false},
		{"Empty argument", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := sanitizeArgument(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected an error, but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if output != tt.expectedOutput {
				t.Errorf("Expected output %q, but got %q", tt.expectedOutput, output)
			}
		})
	}
}

func TestSplitWindowsCommand(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput []string
	}{
		{"Simple command", "dir C:\\Users", []string{"dir", "C:\\Users"}},
		{"Command with quotes", `echo "Hello World"`, []string{"echo", `"Hello World"`}},
		{"Complex command", `dir "C:\Program Files" /s`, []string{"dir", `"C:\Program Files"`, "/s"}},
		{"Command with multiple spaces", "ping   google.com", []string{"ping", "google.com"}},
		{"Command with quoted path", `dir "C:\Program Files" /s`, []string{"dir", `"C:\Program Files"`, "/s"}},
		{"Echo with quotes", `echo "Hello World"`, []string{"echo", `"Hello World"`}},
		{"Copy with quoted filename", `copy file1.txt "file 2.txt"`, []string{"copy", "file1.txt", `"file 2.txt"`}},
		{"Simple command", "ping localhost", []string{"ping", "localhost"}},
		{"Command with multiple quoted parts", `echo "First part" middle "Last part"`, []string{"echo", `"First part"`, "middle", `"Last part"`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := splitWindowsCommand(tt.input)

			if len(output) != len(tt.expectedOutput) {
				t.Errorf("Expected %d parts, but got %d", len(tt.expectedOutput), len(output))
			}
			for i := range output {
				if output[i] != tt.expectedOutput[i] {
					t.Errorf("Part %d: expected %q, but got %q", i+1, tt.expectedOutput[i], output[i])
				}
			}
		})
	}
}
