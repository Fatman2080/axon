package main

import "strings"

func buildDispatchCommand(template string, privateKey string, publicKey string, agentVaultAddr string) string {
	cmd := strings.ReplaceAll(template, "#prikey#", privateKey)
	cmd = strings.ReplaceAll(cmd, "#pubkey#", publicKey)
	cmd = strings.ReplaceAll(cmd, "#agentvaultaddr#", agentVaultAddr)
	return cmd
}
