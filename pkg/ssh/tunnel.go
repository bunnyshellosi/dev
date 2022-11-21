package ssh

import (
	"io"
	"log"
	"net"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

type SSHTunnel struct {
	SSHServerEndpoint *Endpoint
	LocalEndpoint     *Endpoint
	RemoteEndpoint    *Endpoint

	Mode ForwardMode

	Config *ssh.ClientConfig

	Logger *log.Logger

	ReadyChannel chan bool
	StopChannel  chan bool

	sshConn  *ssh.Client
	listener net.Listener
}

func (tunnel *SSHTunnel) logf(fmt string, args ...interface{}) {
	if tunnel.Logger != nil {
		tunnel.Logger.Printf(fmt, args...)
	}
}

func (tunnel *SSHTunnel) Start() error {
	if err := tunnel.setupSSHConnection(); err != nil {
		return err
	}

	if err := tunnel.listen(); err != nil {
		return err
	}

	if tunnel.LocalEndpoint.Port == 0 && tunnel.Mode == ForwardModeForward {
		tunnel.LocalEndpoint.Port = tunnel.listener.Addr().(*net.TCPAddr).Port
	} else if tunnel.RemoteEndpoint.Port == 0 && tunnel.Mode == ForwardModeReverse {
		tunnel.RemoteEndpoint.Port = tunnel.listener.Addr().(*net.TCPAddr).Port
	}

	return nil
}

func (tunnel *SSHTunnel) Wait() {
	<-tunnel.StopChannel
}

func (tunnel *SSHTunnel) Run() error {
	if err := tunnel.Start(); err != nil {
		return err
	}

	if tunnel.ReadyChannel != nil {
		close(tunnel.ReadyChannel)
	}

	tunnel.Wait()

	return nil
}

func (tunnel *SSHTunnel) setupSSHConnection() error {
	sshConn, err := ssh.Dial("tcp", tunnel.SSHServerEndpoint.String(), tunnel.Config)
	if err != nil {
		return err
	}

	tunnel.sshConn = sshConn

	return nil
}

func (tunnel *SSHTunnel) listen() error {
	var listener net.Listener
	var err error
	switch tunnel.Mode {
	case ForwardModeForward:
		listener, err = net.Listen("tcp", tunnel.LocalEndpoint.String())
	case ForwardModeReverse:
		listener, err = tunnel.sshConn.Listen("tcp", tunnel.RemoteEndpoint.String())
	}
	if err != nil {
		return err
	}

	tunnel.listener = listener
	go tunnel.waitForConnection()

	return nil
}

func (tunnel *SSHTunnel) waitForConnection() {
	for {
		conn, err := tunnel.listener.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				tunnel.logf("error on listener.Accept: %s", err)
			}
			return
		}

		go tunnel.handleConnection(conn)
	}
}

func (tunnel *SSHTunnel) handleConnection(bindConn net.Conn) {
	defer bindConn.Close()

	var dialConn net.Conn
	var err error
	switch tunnel.Mode {
	case ForwardModeForward:
		dialConn, err = tunnel.sshConn.Dial("tcp", tunnel.RemoteEndpoint.String())
	case ForwardModeReverse:
		dialConn, err = net.Dial("tcp", tunnel.LocalEndpoint.String())
	}
	if err != nil {
		tunnel.logf("remote dial error: %s", err)
		return
	}
	defer dialConn.Close()

	var wg sync.WaitGroup
	copyConn := func(writer, reader net.Conn) {
		defer wg.Done()
		_, err := io.Copy(writer, reader)
		if err != nil {
			tunnel.logf("io.Copy error: %s", err)
		}
	}

	wg.Add(1)
	go copyConn(bindConn, dialConn)

	wg.Add(1)
	go copyConn(dialConn, bindConn)

	wg.Wait()
}

func (tunnel *SSHTunnel) Stop() {
	if tunnel.listener != nil {
		tunnel.listener.Close()
	}

	if tunnel.sshConn != nil {
		tunnel.sshConn.Close()
	}

	if tunnel.StopChannel != nil {
		close(tunnel.StopChannel)
	}
}

func NewSSHTunnel() *SSHTunnel {
	return &SSHTunnel{
		Config: &ssh.ClientConfig{
			Auth:            []ssh.AuthMethod{},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
		Logger: nil,
		Mode:   ForwardModeForward,

		ReadyChannel: make(chan bool),
		StopChannel:  make(chan bool),
	}
}

func (tunnel *SSHTunnel) WithAuths(values ...ssh.AuthMethod) *SSHTunnel {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithAuths")
		}
		tunnel.Config.Auth = append(tunnel.Config.Auth, values[i])
	}
	return tunnel
}

func (tunnel *SSHTunnel) WithSSHServerEndpoint(endpoint *Endpoint) *SSHTunnel {
	tunnel.SSHServerEndpoint = endpoint
	tunnel.Config.User = endpoint.User
	return tunnel
}

func (tunnel *SSHTunnel) WithLocalEndpoint(endpoint *Endpoint) *SSHTunnel {
	tunnel.LocalEndpoint = endpoint
	return tunnel
}

func (tunnel *SSHTunnel) WithRemoteEndpoint(endpoint *Endpoint) *SSHTunnel {
	tunnel.RemoteEndpoint = endpoint
	return tunnel
}

func (tunnel *SSHTunnel) WithMode(mode ForwardMode) *SSHTunnel {
	tunnel.Mode = mode
	return tunnel
}

func (tunnel *SSHTunnel) WithLog(logger *log.Logger) *SSHTunnel {
	tunnel.Logger = logger
	return tunnel
}
