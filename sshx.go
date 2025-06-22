package sshx

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var ErrNoAuthMethod = errors.New("at least one auth method (signer or password) must be provided")

// sshx Client is a wrapper that includes an SSH client and an SFTP client. It also provides simplified Run, Upload and Download methods.
type Client struct {
	SSHClient  *ssh.Client
	SFTPClient *sftp.Client
}

type clientOptions struct {
	signer         ssh.Signer
	password       string
	setup          bool
	additionalKeys []ssh.PublicKey
	network        string
}

func newOptions(opts ...ClientOption) *clientOptions {
	o := &clientOptions{
		network: "tcp",
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type ClientOption func(*clientOptions)

// WithSigner sets the SSH key on the sshxClient.
func WithSigner(signer ssh.Signer) ClientOption {
	return func(o *clientOptions) {
		o.signer = signer
	}
}

// WithPassword sets the password on the sshxClient.
// Please consider using WithSigner instead.
// If you are using the password for a first time setup, use WithFirstTimeSetup as well.
func WithPassword(password string) ClientOption {
	return func(o *clientOptions) {
		o.password = password
	}
}

// WithFirstTimeSetup uses the password (if set) to authenticate and the signer's public key along with any additional SSH keys.
// After this, the password is removed and the signer is used to authenticate. If the password is not set, the signer is used to authenticate
// and only the additional SSH keys are added to the server. Using this method multiple times will result in duplicate keys on the server.
func WithFirstTimeSetup(additionalKeys ...ssh.PublicKey) ClientOption {
	return func(o *clientOptions) {
		o.setup = true
		o.additionalKeys = additionalKeys
	}
}

// WithNetwork sets the network to use for SSH connections.
// Defaults to "tcp".
func WithNetwork(network string) ClientOption {
	return func(o *clientOptions) {
		o.network = network
	}
}

// NewClient creates a new sshx Client with the given IP, username, and options.
// At least one of the signer or password must be set.
func NewClient(addr string, username string, options ...ClientOption) (*Client, error) {
	opts := newOptions(options...)

	if opts.signer == nil && opts.password == "" {
		return nil, ErrNoAuthMethod
	}

	config := &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if opts.setup {
		err := addKeyToServer(opts.network, addr, username, opts.signer, opts.password, opts.additionalKeys...)
		if err != nil {
			return nil, err
		}
	}

	if opts.password != "" {
		config.Auth = []ssh.AuthMethod{ssh.Password(opts.password)}
	}

	if opts.signer != nil {
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(opts.signer)}
	}

	sshClient, err := ssh.Dial(opts.network, addr, config)
	if err != nil {
		return nil, err
	}

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, err
	}

	return &Client{
		sshClient,
		sftpClient,
	}, nil
}

// addKeyToServer adds the signer's public key and any additional SSH keys to the server at the given IP using the provided username and password.
func addKeyToServer(network, addr, username string, signer ssh.Signer, tempPassword string, additionalKeys ...ssh.PublicKey) error {
	opts := []ClientOption{
		WithPassword(tempPassword),
		WithNetwork(network),
	}

	if tempPassword == "" {
		opts = append(opts, WithSigner(signer))
	}

	client, err := NewClient(addr, username, opts...)
	if err != nil {
		return err
	}

	keys := ""
	if signer != nil && tempPassword != "" {
		keys = string(ssh.MarshalAuthorizedKey(signer.PublicKey()))
	}
	for _, ak := range additionalKeys {
		keys += string(ssh.MarshalAuthorizedKey(ak))
	}

	if err := client.SFTPClient.MkdirAll(".ssh"); err != nil {
		return err
	}

	if err := client.Upload(strings.NewReader(keys), ".ssh/authorized_keys", false); err != nil {
		return err
	}

	return nil
}

// Run runs the given command on the remote server in a new session.
func (c *Client) Run(cmd string) (string, error) {
	sess, err := c.SSHClient.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	out, err := sess.CombinedOutput(cmd)
	return string(out), err
}

// Upload uploads the src reader to the remote path on the server.
func (c *Client) Upload(src io.Reader, remotePath string, append bool) error {
	flags := os.O_WRONLY | os.O_CREATE
	if append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	dst, err := c.SFTPClient.OpenFile(remotePath, flags)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = dst.ReadFrom(src)
	if err != nil {
		return err
	}

	return nil
}

// Download downloads a file from the remote server to the dst writer.
func (c *Client) Download(dst io.Writer, remotePath string) error {
	src, err := c.SFTPClient.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = src.WriteTo(dst)
	if err != nil {
		return err
	}
	return nil
}

// Close closes the SSH and SFTP clients.
func (c *Client) Close() error {
	return errors.Join(
		c.SSHClient.Close(),
		c.SFTPClient.Close(),
	)
}
