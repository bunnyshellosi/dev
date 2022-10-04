//go:build !windows
// +build !windows

package ssh

func processConfigPathForInclude(path string) string {
	return path
}
