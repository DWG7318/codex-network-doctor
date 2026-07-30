package oneclick

import (
	"path/filepath"
	"strings"
	"time"
)

func Arguments(outputDirectory, hostname string, now time.Time) ([]string, string) {
	safeHost := sanitizeHostname(hostname)
	if safeHost == "" {
		safeHost = "unknown-host"
	}
	filename := "cnd诊断结果-" + safeHost + "-" + now.Format("20060102-150405") + ".zip"
	output := filepath.Join(outputDirectory, filename)
	return []string{"collect", "--provider", "powershell", "--output", output}, output
}

func OutputDirectory(desktopResolver func() (string, error), executablePath string) string {
	desktop, err := desktopResolver()
	if err == nil && filepath.IsAbs(desktop) {
		return desktop
	}
	return filepath.Dir(executablePath)
}

func sanitizeHostname(hostname string) string {
	var builder strings.Builder
	pendingSeparator := false
	for _, character := range hostname {
		allowed := character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			character == '_' || character == '.' || character == '-'
		if !allowed {
			if builder.Len() > 0 {
				pendingSeparator = true
			}
			continue
		}
		if pendingSeparator && character != '-' && character != '.' && character != '_' {
			builder.WriteByte('-')
		}
		pendingSeparator = false
		builder.WriteRune(character)
	}
	return strings.Trim(builder.String(), "._-")
}
