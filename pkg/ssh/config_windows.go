package ssh

import "strings"

func processConfigPathForInclude(path string) string {
	return "/" + strings.ReplaceAll(
		strings.ReplaceAll(path, "\\", "/"),
		":",
		"",
	)
}
