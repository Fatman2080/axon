package main

import (
	"regexp"
	"strings"
)

// shellSafeValue ensures a value only contains safe characters (hex, alphanumeric, 0x prefix).
// Rejects anything with shell metacharacters.
var shellSafePattern = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

func shellEscapeValue(val string) string {
	// If the value is purely alphanumeric/hex (which keys and addresses always are), pass through.
	if shellSafePattern.MatchString(val) {
		return val
	}
	// Otherwise single-quote escape: replace ' with '\'' and wrap in single quotes.
	return "'" + strings.ReplaceAll(val, "'", `'\''`) + "'"
}

func buildDispatchCommand(template string, privateKey string, publicKey string, agentVaultAddr string) string {
	cmd := strings.ReplaceAll(template, "#prikey#", shellEscapeValue(privateKey))
	cmd = strings.ReplaceAll(cmd, "#pubkey#", shellEscapeValue(publicKey))
	cmd = strings.ReplaceAll(cmd, "#agentvaultaddr#", shellEscapeValue(agentVaultAddr))
	return cmd
}
