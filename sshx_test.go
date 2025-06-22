package sshx_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/pedramktb/go-sshx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/ssh"
)

func testContainer(ctx context.Context) (container testcontainers.Container, addr string) {
	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "./",
			Dockerfile: "sshx_test.dockerfile",
		},
		ExposedPorts: []string{"22/tcp"},
		WaitingFor:   wait.ForListeningPort("22/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		panic(err)
	}

	natPort, err := container.MappedPort(ctx, "22")
	if err != nil {
		panic(err)
	}

	return container, fmt.Sprintf("%s:%d", host, natPort.Int())
}

func Test_FirstTimeSetup(t *testing.T) {
	ctx := context.Background()
	container, addr := testContainer(ctx)
	defer func() {
		_ = container.Terminate(ctx)
	}()

	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = sshx.NewClient(
		addr,
		"root",
		sshx.WithPassword("test"),
		sshx.WithSigner(signer),
		sshx.WithFirstTimeSetup(),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = sshx.NewClient(
		addr,
		"root",
		sshx.WithSigner(signer),
	)
	if err != nil {
		t.Fatal(err)
	}
}
