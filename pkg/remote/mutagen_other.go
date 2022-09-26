//go:build !windows
// +build !windows

package remote

func getMutagenBinFilename() string {
	return mutagenBinFilename
}
