package ssh

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"
)

const (
	SSHAuthSockEnvVar = "SSH_AUTH_SOCK"
)

type SSHTerminal struct {
	Server *Endpoint
	Config *ssh.ClientConfig

	ReadyChannel chan bool
}

func NewSSHTerminal(host string, port int, auth ssh.AuthMethod) *SSHTerminal {
	server := NewEndpoint(host, port)

	return &SSHTerminal{
		Config: &ssh.ClientConfig{
			User: server.User,
			Auth: []ssh.AuthMethod{auth},
			HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return nil
			},
		},
		Server: server,

		ReadyChannel: make(chan bool),
	}
}

func (sshTerminal *SSHTerminal) Start() error {
	serverConn, err := ssh.Dial("tcp", sshTerminal.Server.String(), sshTerminal.Config)
	if err != nil {
		return err
	}
	defer serverConn.Close()

	if err := showMotd(serverConn); err != nil {
		return err
	}

	session, err := makeSession(serverConn)
	if err != nil {
		return err
	}

	// try forwarding the SSH agent
	sshAuthSock := os.Getenv(SSHAuthSockEnvVar)
	if sshAuthSock != "" {
		agent.ForwardToRemote(serverConn, sshAuthSock)
		agent.RequestAgentForwarding(session)
	}

	termFd := int(os.Stdout.Fd())
	if !term.IsTerminal(termFd) {
		return fmt.Errorf("no terminal available")
	}

	oldState, err := makeRawTerminal(termFd)
	if err != nil {
		return err
	}
	if oldState != nil {
		defer term.Restore(termFd, oldState)
	}

	w, h, err := term.GetSize(termFd)
	if err != nil {
		return err
	}

	terminalModes := ssh.TerminalModes{
		ssh.ECHO:  0,
		ssh.IGNCR: 1,
	}
	if err := session.RequestPty("xterm-256color", h, w, terminalModes); err != nil {
		return err
	}

	if err := session.Shell(); err != nil {
		return err
	}

	if sshTerminal.ReadyChannel != nil {
		close(sshTerminal.ReadyChannel)
	}

	return session.Wait()
}

func makeSession(client *ssh.Client) (*ssh.Session, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	session.Stdin, session.Stdout, session.Stderr = stdStreams()

	return session, nil
}

// This should be moved to a ssh-server banner
func showMotd(client *ssh.Client) error {
	session, err := makeSession(client)
	if err != nil {
		return err
	}

	return session.Run("cat /opt/bunnyshell/motd.txt")
}
