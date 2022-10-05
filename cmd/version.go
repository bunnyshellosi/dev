package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"bunnyshell.com/dev/pkg/build"
)

func init() {
	var showAll bool

	var command = &cobra.Command{
		Use:   "version",
		Short: "Version Information",
		Run: func(cmd *cobra.Command, args []string) {
			if build.Version == "dev" {
				cmd.Println("You are using a development version")
			} else {
				currentVersion := "v" + build.Version
				cmd.Printf("%s version: %s-%s\n", build.Name, currentVersion, build.Commit)

				latestRelease, _ := getLatestRelease()
				if latestRelease != "" && semver.Compare(currentVersion, latestRelease) < 0 {
					fmt.Printf("Your version %s is older than the latest: %s\n", currentVersion, latestRelease)
				}
			}

			if showAll {
				cmd.Printf("SSHServer version: %s\n", build.SSHServerVersion)
				cmd.Printf("Mutagen version: %s\n", build.MutagenVersion)
			}
		},
	}

	command.Flags().BoolVarP(&showAll, "show-all", "a", false, "also display SSHServer and Mutagen versions")

	rootCmd.AddCommand(command)
}

func getLatestRelease() (string, error) {
	// Set up the HTTP request
	req, err := http.NewRequest("GET", build.LatestReleaseUrl, nil)
	if err != nil {
		return "", err
	}

	transport := http.Transport{}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return "", err
	}
	// Check if you received the status codes you expect. There may
	// status codes other than 200 which are acceptable.
	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		return "", errors.New("must be a redirect")
	}

	redirect := resp.Header.Get("Location")
	parts := strings.Split(redirect, "/")
	return parts[len(parts)-1], nil
}
