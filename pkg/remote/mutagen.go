package remote

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"bunnyshell.com/dev/pkg/build"
	mutagenConfig "bunnyshell.com/dev/pkg/mutagen/config"
	"bunnyshell.com/dev/pkg/util"
	"gopkg.in/yaml.v3"
)

const (
	mutagenBinFilename      = "mutagen"
	mutagenDownloadFilename = "mutagen_%s_%s_%s.tar.gz"
	mutagenDownloadUrl      = "https://github.com/mutagen-io/mutagen/releases/download/%s/%s"

	mutagenConfigFilenamePattern = "mutagen.%s.yaml"
	mutagenIgnoreFilename        = ".rdignore"
)

func (r *RemoteDevelopment) ensureMutagen() error {
	r.StartSpinner(" Setup Mutagen")
	defer r.StopSpinner()

	if err := ensureMutagenBin(); err != nil {
		return err
	}

	return r.ensureMutagenConfigFile()
}

func (r *RemoteDevelopment) ensureMutagenConfigFile() error {
	mutagenConfigFilePath, err := r.getMutagenConfigFilePath()
	if err != nil {
		return err
	}

	enableVCS := true
	sessionIgnores, err := r.getMutagenSessionIgnores()
	if sessionIgnores == nil {
		r.StopSpinner()
		fmt.Printf("INFO: All files will be synchronized. You can exclude files from sync by creating a %s/%s file.\n", r.localSyncPath, mutagenIgnoreFilename)
		r.StartSpinner("")
	}
	if err != nil {
		return err
	}
	ignore := mutagenConfig.NewIgnore().WithVCS(&enableVCS).WithPaths(sessionIgnores)
	defaults := mutagenConfig.NewSyncDefaults().WithMode(mutagenConfig.TwoWayResolved).WithIgnore(ignore)
	sync := mutagenConfig.NewSync().WithDefaults(defaults)
	config := mutagenConfig.NewConfiguration().WithSync(sync)

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(mutagenConfigFilePath, data, 0644)
}

func (r *RemoteDevelopment) startMutagenSession() error {
	r.StartSpinner(" Start Mutagen Session")
	defer r.StopSpinner()

	mutagenBinPath, err := getMutagenBinPath()
	if err != nil {
		return err
	}
	mutagenConfigFilePath, err := r.getMutagenConfigFilePath()
	if err != nil {
		return err
	}

	hostname, err := r.getSSHHostname()
	if err != nil {
		return err
	}
	sessionName, err := r.getMutagenSessionName()
	if err != nil {
		return err
	}
	mutagenArgs := []string{
		"sync",
		"create",
		"-n", sessionName,
		"--no-global-configuration",
		"-c", mutagenConfigFilePath,
		r.localSyncPath,
		fmt.Sprintf(
			"%s:%s",
			hostname,
			r.remoteSyncPath,
		),
	}

	mutagenCmd := exec.Command(mutagenBinPath, mutagenArgs...)

	output, err := mutagenCmd.CombinedOutput()
	if mutagenCmd.ProcessState.ExitCode() != 0 {
		fmt.Println(string(output))
	}

	return err
}

func (r *RemoteDevelopment) getMutagenSessionIgnores() ([]string, error) {
	ignoreFilePath := filepath.Join(r.localSyncPath, mutagenIgnoreFilename)
	if _, err := os.Stat(ignoreFilePath); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	readFile, err := os.Open(ignoreFilePath)
	if err != nil {
		return nil, err
	}
	defer readFile.Close()

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	ignores := []string{}
	for fileScanner.Scan() {
		ignorePath := strings.TrimSpace(fileScanner.Text())
		if ignorePath == "" || strings.HasPrefix(ignorePath, "#") {
			continue
		}
		ignores = append(ignores, ignorePath)
	}

	return ignores, nil
}

func (r *RemoteDevelopment) terminateMutagenSession() error {
	mutagenBinPath, err := getMutagenBinPath()
	if err != nil {
		return err
	}

	sessionName, err := r.getMutagenSessionName()
	if err != nil {
		return err
	}

	mutagenArgs := []string{
		"sync",
		"terminate",
		sessionName,
	}

	mutagenCmd := exec.Command(mutagenBinPath, mutagenArgs...)
	mutagenCmd.Run()

	return nil
}

func (r *RemoteDevelopment) terminateMutagenDaemon() error {
	mutagenBinPath, err := getMutagenBinPath()
	if err != nil {
		return err
	}

	mutagenArgs := []string{
		"daemon",
		"stop",
	}

	mutagenCmd := exec.Command(mutagenBinPath, mutagenArgs...)
	mutagenCmd.Run()

	return nil
}

func (r *RemoteDevelopment) getMutagenSessionName() (string, error) {
	sessionKey, err := r.getMutagenSessionKey()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("rd-%s", sessionKey), nil
}

func (r *RemoteDevelopment) getMutagenSessionKey() (string, error) {
	resource, err := r.getResource()
	if err != nil {
		return "", err
	}

	plaintext := fmt.Sprintf("%s-%s-%s", r.remoteSyncPath, resource.GetName(), resource.GetNamespace())
	hash := md5.Sum([]byte(plaintext))
	return hex.EncodeToString(hash[:])[:16], nil
}

func getMutagenBinPath() (string, error) {
	workspaceDir, err := util.GetRemoteDevWorkspaceDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(workspaceDir, getMutagenBinFilename()), nil
}

func (r *RemoteDevelopment) getMutagenConfigFilePath() (string, error) {
	workspaceDir, err := util.GetRemoteDevWorkspaceDir()
	if err != nil {
		return "", err
	}

	sessionKey, err := r.getMutagenSessionKey()
	if err != nil {
		return "", err
	}

	return filepath.Join(workspaceDir, fmt.Sprintf(mutagenConfigFilenamePattern, sessionKey)), nil
}

func ensureMutagenBin() error {
	mutagenBinPath, err := getMutagenBinPath()
	if err != nil {
		return err
	}

	stats, err := os.Stat(mutagenBinPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err == nil && stats.Size() > 0 && !stats.IsDir() {
		return nil
	}

	downloadFilename := fmt.Sprintf(mutagenDownloadFilename, runtime.GOOS, runtime.GOARCH, build.MutagenVersion)
	mutagenArchivePath := filepath.Join(filepath.Dir(mutagenBinPath), downloadFilename)
	downloadUrl := fmt.Sprintf(mutagenDownloadUrl, build.MutagenVersion, downloadFilename)

	err = downloadMutagenArchive(downloadUrl, mutagenArchivePath)
	if err != nil {
		return err
	}

	err = extractMutagenBin(mutagenArchivePath, mutagenBinPath)
	if err != nil {
		return err
	}

	return removeMutagenArchive(mutagenArchivePath)
}

func removeMutagenArchive(filePath string) error {
	return os.Remove(filePath)
}

func downloadMutagenArchive(source, destination string) error {
	// Configure the connection timeout
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 60 * time.Second,
		}).DialContext,
	}

	client := &http.Client{
		Transport: transport,
	}

	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := client.Get(source)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractMutagenBin(source, destination string) error {
	return extractMutagenBinTarGz(source, destination)
}

func extractMutagenBinTarGz(source, destination string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	gzipReader, err := gzip.NewReader(sourceFile)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if header.Name == getMutagenBinFilename() {
			destinationFile, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			defer destinationFile.Close()

			if _, err := io.Copy(destinationFile, tarReader); err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}
