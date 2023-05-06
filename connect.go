package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	UserName   string
	Password   string
	Addr       string
	PrivateKey string
	Timeout    time.Duration
}

func SSH(config Config) (*ssh.Client, error) {
	var (
		auth         []ssh.AuthMethod
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		err          error
	)

	if config.Password == "" {

		var pemBytes []byte

		if config.PrivateKey == "" {
			// read private key file
			var homeDir, err = os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("get home dir failed %v", err)
			}
			var privateKeyPath = homeDir + "/.ssh/id_rsa"
			pemBytes, err = os.ReadFile(privateKeyPath)
			if err != nil {
				return nil, fmt.Errorf("reading private key file failed %v", err)
			}
		} else {
			var absPath, err = filepath.Abs(config.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("get abs path failed %v", err)
			}
			pemBytes, err = os.ReadFile(absPath)
			if err != nil {
				return nil, fmt.Errorf("reading private key file failed %v", err)
			}
		}

		// create signer
		// generate signer instance from plain key
		signer, err := ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, fmt.Errorf("parsing plain private key failed %v", err)
		}

		auth = append(auth, ssh.PublicKeys(signer))
	} else {
		// get auth method
		auth = append(auth, ssh.Password(config.Password))
	}

	clientConfig = &ssh.ClientConfig{
		User:    config.UserName,
		Auth:    auth,
		Timeout: config.Timeout,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	if client, err = ssh.Dial("tcp", config.Addr, clientConfig); err != nil {
		return nil, err
	}

	return client, nil
}

func NewSession(config Config) (*ssh.Session, error) {
	client, err := SSH(config)
	if err != nil {
		return nil, err
	}
	return client.NewSession()
}
