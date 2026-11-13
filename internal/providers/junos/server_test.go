package junos

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
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
	cmdVersion:      "ex_version.xml",
	cmdPorts:        "ex_terse.xml",
	cmdDescriptions: "ex_descriptions.xml",
	cmdNeighbors:    "ex_lldp.xml",
}

func newFakeDevice(t *testing.T, replies map[string]string) *fakeDevice {
	t.Helper()
	return startFakeDevice(t, replies, false)
}

// newOldFakeDevice only speaks what Junos 12.x offers: an ssh-rsa host key,
// the group14-sha1 key exchange and the aes128-cbc cipher.
func newOldFakeDevice(t *testing.T, replies map[string]string) *fakeDevice {
	t.Helper()
	return startFakeDevice(t, replies, true)
}

func startFakeDevice(t *testing.T, replies map[string]string, old bool) *fakeDevice {
	t.Helper()
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "client_key")
	client, err := sshkey.LoadOrCreate(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	hostSigner := newHostKey(t, old)
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
	if old {
		config.KeyExchanges = []string{ssh.InsecureKeyExchangeDH14SHA1}
		config.Ciphers = []string{ssh.InsecureCipherAES128CBC}
		config.MACs = []string{ssh.HMACSHA1}
	}

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

func newHostKey(t *testing.T, old bool) ssh.Signer {
	t.Helper()
	if !old {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		signer, err := ssh.NewSignerFromKey(priv)
		if err != nil {
			t.Fatal(err)
		}
		return signer
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	sha1Only, err := ssh.NewSignerWithAlgorithms(signer.(ssh.AlgorithmSigner), []string{ssh.KeyAlgoRSA})
	if err != nil {
		t.Fatal(err)
	}
	return sha1Only
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
