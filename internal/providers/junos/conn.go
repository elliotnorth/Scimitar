package junos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	dialTimeout    = 10 * time.Second
	commandTimeout = 30 * time.Second
	maxOutput      = 8 << 20
)

// Host key types in the order Scimitar asks for them. The order is fixed so the
// key fingerprinted at setup is the same one offered on every later connection.
// Plain ssh-rsa comes last for Junos releases older than 15.1, which offer nothing else.
var hostKeyAlgorithms = []string{
	ssh.KeyAlgoED25519,
	ssh.KeyAlgoECDSA256,
	ssh.KeyAlgoECDSA384,
	ssh.KeyAlgoECDSA521,
	ssh.KeyAlgoRSASHA512,
	ssh.KeyAlgoRSASHA256,
	ssh.KeyAlgoRSA,
}

// withOlderJunos adds the SHA-1 key exchanges and the CBC cipher that Junos 12.x
// still needs, after every modern choice so they are only used when nothing
// else is offered. The pinned host key still rules out impersonation.
func withOlderJunos(config *ssh.ClientConfig) *ssh.ClientConfig {
	supported := ssh.SupportedAlgorithms()
	config.HostKeyAlgorithms = hostKeyAlgorithms
	config.KeyExchanges = append(supported.KeyExchanges, ssh.InsecureKeyExchangeDH14SHA1, ssh.InsecureKeyExchangeDHGEXSHA1)
	config.Ciphers = append(supported.Ciphers, ssh.InsecureCipherAES128CBC)
	config.MACs = supported.MACs
	return config
}

var errOutputTooLarge = errors.New("output too large")

// conn keeps one SSH connection open and runs each command in a new session on it.
type conn struct {
	addr   string
	config *ssh.ClientConfig

	mu     sync.Mutex
	client *ssh.Client
}

func pinnedHostKey(want string) ssh.HostKeyCallback {
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		if got := ssh.FingerprintSHA256(key); got != want {
			return fmt.Errorf("host key changed: expected %s, got %s", want, got)
		}
		return nil
	}
}

func dial(ctx context.Context, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	d := net.Dialer{Timeout: dialTimeout}
	netConn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(dialTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	netConn.SetDeadline(deadline)
	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, config)
	if err != nil {
		netConn.Close()
		return nil, err
	}
	netConn.SetDeadline(time.Time{})
	return ssh.NewClient(sshConn, chans, reqs), nil
}

func (c *conn) get(ctx context.Context) (*ssh.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		return c.client, nil
	}
	client, err := dial(ctx, c.addr, c.config)
	if err != nil {
		return nil, fmt.Errorf("junos: connect %s: %w", c.addr, err)
	}
	c.client = client
	return client, nil
}

func (c *conn) drop(client *ssh.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == client {
		c.client.Close()
		c.client = nil
	}
}

func (c *conn) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
}

// run executes one CLI command and returns its output. A broken connection is
// replaced once; a command that outlives its timeout drops the connection, since
// a stuck session can leave it unusable.
func (c *conn) run(ctx context.Context, command string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()
	return c.try(ctx, command, true)
}

type result struct {
	out       []byte
	err       error
	noSession bool
}

func (c *conn) try(ctx context.Context, command string, retry bool) ([]byte, error) {
	client, err := c.get(ctx)
	if err != nil {
		return nil, err
	}
	done := make(chan result, 1)
	go func() { done <- runSession(client, command) }()
	select {
	case r := <-done:
		if r.noSession {
			c.drop(client)
			if retry {
				return c.try(ctx, command, false)
			}
			return nil, fmt.Errorf("junos: open session: %w", r.err)
		}
		if r.err != nil {
			return nil, fmt.Errorf("junos: %s: %w", command, r.err)
		}
		return r.out, nil
	case <-ctx.Done():
		c.drop(client)
		return nil, fmt.Errorf("junos: %s: %w", command, ctx.Err())
	}
}

func runSession(client *ssh.Client, command string) result {
	session, err := client.NewSession()
	if err != nil {
		return result{err: err, noSession: true}
	}
	defer session.Close()
	stdout := &limitedBuffer{max: maxOutput}
	stderr := &limitedBuffer{max: 64 << 10}
	session.Stdout = stdout
	session.Stderr = stderr
	err = session.Run(command)
	var missing *ssh.ExitMissingError
	if errors.As(err, &missing) {
		err = nil
	}
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			err = fmt.Errorf("%w: %s", err, firstLine(msg))
		}
		return result{err: err}
	}
	return result{out: stdout.Bytes()}
}

type limitedBuffer struct {
	bytes.Buffer
	max int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.max {
		return 0, errOutputTooLarge
	}
	return b.Buffer.Write(p)
}

// FetchHostKey connects just far enough to read the device's host key and
// returns its SHA256 fingerprint and key type, for the user to confirm.
func FetchHostKey(ctx context.Context, address string) (fingerprint, keyType string, err error) {
	addr, err := NormalizeAddress(address)
	if err != nil {
		return "", "", err
	}
	var key ssh.PublicKey
	errCaptured := errors.New("host key captured")
	config := withOlderJunos(&ssh.ClientConfig{
		User: "scimitar",
		HostKeyCallback: func(_ string, _ net.Addr, k ssh.PublicKey) error {
			key = k
			return errCaptured
		},
	})
	client, err := dial(ctx, addr, config)
	if client != nil {
		client.Close()
	}
	if key == nil {
		return "", "", fmt.Errorf("could not reach %s: %w", addr, err)
	}
	return ssh.FingerprintSHA256(key), key.Type(), nil
}
