package junos

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/elliotnorth/Scimitar/internal/sshkey"
)

// fakeDevice is an SSH server that answers exec requests with fixture files, like a Junos CLI.
type fakeDevice struct {
	addr     string
	hostKey  string
	keyFile  string
	username string

	mu       sync.Mutex
	commands []string
	logins   int
	replies  map[string]string
}

var exReplies = map[string]string{
	cmdVersion:      "ex_version.json",
	cmdPorts:        "ex_terse.json",
	cmdDescriptions: "ex_descriptions.json",
	cmdNeighbors:    "ex_lldp.json",
}

func newFakeDevice(t *testing.T, replies map[string]string) *fakeDevice {
	t.Helper()
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "client_key")
	client, err := sshkey.LoadOrCreate(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatal(err)
	}
	dev := &fakeDevice{hostKey: ssh.FingerprintSHA256(hostSigner.PublicKey()), keyFile: keyFile, username: "scimitar", replies: replies}

	allowed := client.PublicKey().Marshal()
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if meta.User() == dev.username && bytes.Equal(key.Marshal(), allowed) {
				return nil, nil
			}
			return nil, errDenied
		},
	}
	config.AddHostKey(hostSigner)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	dev.addr = listener.Addr().String()
	go func() {
		for {
			nc, err := listener.Accept()
			if err != nil {
				return
			}
			go dev.serve(nc, config)
		}
	}()
	return dev
}

type denied struct{}

func (denied) Error() string { return "denied" }

var errDenied = denied{}

func (d *fakeDevice) serve(nc net.Conn, config *ssh.ServerConfig) {
	conn, chans, reqs, err := ssh.NewServerConn(nc, config)
	if err != nil {
		nc.Close()
		return
	}
	defer conn.Close()
	d.mu.Lock()
	d.logins++
	d.mu.Unlock()
	go ssh.DiscardRequests(reqs)
	for nch := range chans {
		if nch.ChannelType() != "session" {
			nch.Reject(ssh.UnknownChannelType, "only sessions")
			continue
		}
		ch, requests, err := nch.Accept()
		if err != nil {
			return
		}
		go d.session(ch, requests)
	}
}

func (d *fakeDevice) session(ch ssh.Channel, requests <-chan *ssh.Request) {
	defer ch.Close()
	for req := range requests {
		if req.Type != "exec" {
			req.Reply(false, nil)
			continue
		}
		var payload struct{ Command string }
		if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
			req.Reply(false, nil)
			return
		}
		req.Reply(true, nil)
		d.mu.Lock()
		d.commands = append(d.commands, payload.Command)
		file, ok := d.replies[payload.Command]
		d.mu.Unlock()
		if ok {
			body, _ := os.ReadFile(filepath.Join("testdata", file))
			ch.Write(body)
		} else {
			ch.Write([]byte("error: syntax error\n"))
		}
		ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
		return
	}
}

func (d *fakeDevice) config(name string) Config {
	return Config{Name: name, Address: d.addr, Username: d.username, HostKey: d.hostKey, KeyFile: d.keyFile}
}

func (d *fakeDevice) loginCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.logins
}
