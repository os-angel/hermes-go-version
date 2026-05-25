package skills

import (
	"os"
	"runtime"
)

// CheckPlatform retorna false si el skill no es compatible con el OS actual.
func CheckPlatform(platforms []string) bool {
	return platformMatches(platforms)
}

// CheckEnvVars retorna variables faltantes (no opcionales).
func CheckEnvVars(vars []EnvVar) []string {
	var missing []string
	for _, v := range vars {
		if v.Optional {
			continue
		}
		if os.Getenv(v.Name) == "" {
			missing = append(missing, v.Name)
		}
	}
	return missing
}

// CheckFiles retorna archivos requeridos que no existen.
func CheckFiles(files []string) []string {
	var missing []string
	for _, f := range files {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			missing = append(missing, f)
		}
	}
	return missing
}

// ReadyStatus retorna "AVAILABLE", "SETUP_NEEDED" o "UNSUPPORTED".
func ReadyStatus(s Skill) (status, note string) {
	if !CheckPlatform(s.Platforms) {
		return "UNSUPPORTED", "Este skill no es compatible con " + runtime.GOOS
	}
	missingEnv := CheckEnvVars(s.RequiredEnvVars)
	missingFiles := CheckFiles(s.RequiredFiles)
	if len(missingEnv) > 0 || len(missingFiles) > 0 {
		return "SETUP_NEEDED", "Faltan: env=" + join(missingEnv) + " files=" + join(missingFiles)
	}
	return "AVAILABLE", ""
}

func join(ss []string) string {
	if len(ss) == 0 {
		return "none"
	}
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ","
		}
		result += s
	}
	return result
}
