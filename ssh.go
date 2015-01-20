package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"strings"
)

type successHandler func(pubkey []byte) (string, error)

func startSSHServer(bind string, hostKey ssh.Signer, hd successHandler) error {
	sshConfig := &ssh.ServerConfig{
		NoClientAuth:      false,
		PublicKeyCallback: acceptAnyKey,
	}
	sshConfig.AddHostKey(hostKey)

	socket, err := net.Listen("tcp", bind)
	if err != nil {
		return fmt.Errorf("opening SSH server socket: %s", err)
	}

	go func() {
		defer socket.Close()
		for {
			tcpConn, err := socket.Accept()
			if err != nil {
				fmt.Fprintln(os.Stderr, "accepting connection:", err)
				continue
			}

			go handleTCPConnection(tcpConn, sshConfig, hd)
		}
	}()

	return nil
}

func handleTCPConnection(tcpConn net.Conn, sshConfig *ssh.ServerConfig, hd successHandler) {
	sshConn, channels, requests, err := ssh.NewServerConn(tcpConn, sshConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ssh handshake:", err)
		return
	}
	defer sshConn.Conn.Close()

	go ssh.DiscardRequests(requests)

	for ch := range channels {
		t := ch.ChannelType()
		if t != "session" {
			ch.Reject(ssh.UnknownChannelType, t)
			continue
		}

		channel, requests, err := ch.Accept()
		if err != nil {
			fmt.Fprintln(os.Stderr, "accepting channel:", err)
			continue
		}

		for req := range requests {
			if req.Type == "shell" {
				req.Reply(true, nil)
				pubkey := []byte(sshConn.Permissions.Extensions["pubkey"])
				url, err := hd(pubkey)

				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					fmt.Fprintln(channel.Stderr(), "Sorry, there was an error signing you in :(")
				} else {
					fmt.Fprintln(channel, url)
				}

				break
			} else {
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}

		channel.Close()
	}
}

func readPrivateKey(path string) (ssh.Signer, error) {
	expandedPath := expandPath(path)
	keyBytes, err := ioutil.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %s", err)
	}

	privateKey, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %s", err)
	}

	return privateKey, nil
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~") {
		user, err := user.Current()
		if err == nil {
			return strings.Replace(p, "~", user.HomeDir, 1)
		}
	}
	return p
}

func acceptAnyKey(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	perm := &ssh.Permissions{
		Extensions: map[string]string{"pubkey": string(key.Marshal())},
	}
	return perm, nil
}
