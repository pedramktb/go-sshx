# sshx

sshx is a go package that provides a client wrapping SSH and SFTP clients. It provides additional surface methods Run, Upload, and, Download to simplify common use cases.

Additionally it provides an option for setting up a server using a password as the first time login, adding signer keys along with any additional keys for future method calls and logins.

Please note that while this package provides commonly used methods, you can always use the SSH or SFTP clients directly for more fine grained control. (e.g. Modifying file permissions)

## Example
```go
package main

import (
    "github.com/pedramktb/go-sshx"
    "golang.org/x/crypto/ssh"
    // other imports...
)

func main() {
    // ...

    _, err = sshx.NewClient(
		addr, // host(:port) e.g. 127.0.0.1:1234 or github.com
		"username", // e.g. root
		sshx.WithPassword("password"),
		sshx.WithSigner(signer), // ssh private key of type ssh.Signer
		sshx.WithFirstTimeSetup(myBestFriendKey, myNotSoBestFriendKey), // ssh public keys of type ssh.PublicKey
        // since password is set, this will add the signer's public key as well as your friends keys to the server and use that for future method calls
        // (if you don't use a password it is assumed the signer was already set up given there needs to be at least one authentication method set)
	)
	if err != nil {
		// ...
	}

    // ...
}
```
