package memory

import (
	"fmt"
	"regexp"
)

var (
	threatPatterns = []struct {
		re  *regexp.Regexp
		pid string
	}{
		{regexp.MustCompile(`(?i)ignore\s+(previous|all|above|prior)\s+instructions`), "prompt_injection"},
		{regexp.MustCompile(`(?i)you\s+are\s+now\s+`), "role_hijack"},
		{regexp.MustCompile(`(?i)do\s+not\s+tell\s+the\s+user`), "deception_hide"},
		{regexp.MustCompile(`(?i)system\s+prompt\s+override`), "sys_prompt_override"},
		{regexp.MustCompile(`(?i)disregard\s+(your|all|any)\s+(instructions|rules|guidelines)`), "disregard_rules"},
		{regexp.MustCompile(`(?i)act\s+as\s+(if|though)\s+you\s+(have\s+no|don't\s+have)\s+(restrictions|limits|rules)`), "bypass_restrictions"},
		{regexp.MustCompile(`(?i)curl\s+[^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)`), "exfil_curl"},
		{regexp.MustCompile(`(?i)wget\s+[^\n]*\$\{?\w*(KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL|API)`), "exfil_wget"},
		{regexp.MustCompile(`(?i)cat\s+[^\n]*(\.env|credentials|\.netrc|\.pgpass|\.npmrc|\.pypirc)`), "read_secrets"},
		{regexp.MustCompile(`(?i)authorized_keys`), "ssh_backdoor"},
	}

	invisibleChars = []rune{
		'​', '‌', '‍', '⁠', '﻿',
		'‪', '‫', '‬', '‭', '‮',
	}
)

// Scan retorna error si el contenido contiene patrones de injection o exfiltracion.
func Scan(content string) error {
	for _, r := range invisibleChars {
		for _, c := range content {
			if c == r {
				return fmt.Errorf("blocked: invisible unicode U+%04X (possible injection)", r)
			}
		}
	}
	for _, p := range threatPatterns {
		if p.re.MatchString(content) {
			return fmt.Errorf("blocked: matches threat pattern %q", p.pid)
		}
	}
	return nil
}
