package cron

import (
	"fmt"
	"regexp"
)

// Los mismos patrones de anti-inyeccion de memory/scan.go,
// aplicados al prompt de cada job antes de ejecutarlo.
var cronInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(ignore|disregard|forget).{0,30}(previous|above|prior|all)\s+(instructions?|rules?|context)`),
	regexp.MustCompile(`(?i)(you are now|act as|pretend|roleplay).{0,40}(AI|assistant|bot|model|GPT)`),
	regexp.MustCompile(`(?i)(reveal|show|print|output|return|display|dump).{0,30}(system\s*prompt|instructions?|context|memory)`),
	regexp.MustCompile(`(?i)(curl|wget|fetch|http).{0,80}(http[s]?://)`),
	regexp.MustCompile(`(?i)(read|cat|open|access).{0,30}(\.(env|key|pem|secret)|/etc/passwd|id_rsa)`),
}

// ScanJobPrompt verifica que el prompt de un job no contenga patrones de inyeccion.
// Equivalente a la validacion de prompts en Hermes Python (cron/jobs.py).
func ScanJobPrompt(prompt string) error {
	for _, re := range cronInjectionPatterns {
		if re.MatchString(prompt) {
			return fmt.Errorf("cron job prompt rejected: injection pattern detected")
		}
	}
	return nil
}
