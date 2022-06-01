/**
* @program: yssh
*
* @description:
*
* @author: lemo
*
* @create: 2022-02-11 01:56
**/

package main

import (
	"io"
	"net"
	"time"

	"github.com/lemoyxk/console"
	"golang.org/x/crypto/ssh"
)

func LocalForward(username, password, serverAddr, localAddr, remoteAddr string) error {
	// Setup SSH config (type *ssh.ClientConfig)
	config := &ssh.ClientConfig{
		User:    username,
		Auth:    []ssh.AuthMethod{ssh.Password(password)},
		Timeout: time.Second * 3,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// Setup localListener (type net.Listener)
	localListener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return err
	}

	for {

		// Setup localConn (type net.Conn)
		localConn, err := localListener.Accept()
		if err != nil {
			return err
		}

		go func() {

			// Setup sshClientConn (type *ssh.ClientConn)
			sshClientConn, err := ssh.Dial("tcp", serverAddr, config)
			if err != nil {
				panic(err)
			}

			// Setup sshConn (type net.Conn)
			sshConn, err := sshClientConn.Dial("tcp", remoteAddr)
			if err != nil {
				console.Error(err)
				_ = localConn.Close()
				return
			}

			// Copy localConn.Reader to sshConn.Writer
			go func() {
				_, err = io.Copy(sshConn, localConn)
				if err != nil {
					console.Error(err)
				}
			}()

			// Copy sshConn.Reader to localConn.Writer
			go func() {
				_, err = io.Copy(localConn, sshConn)
				if err != nil {
					console.Error(err)
				}
			}()

		}()
	}

}

func RemoteForward(username, password, serverAddr, remoteAddr, localAddr string) error {
	// Setup SSH config (type *ssh.ClientConfig)
	config := ssh.ClientConfig{
		User:    username,
		Auth:    []ssh.AuthMethod{ssh.Password(password)},
		Timeout: time.Second * 3,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// Setup sshClientConn (type *ssh.ClientConn)
	sshClientConn, err := ssh.Dial("tcp", serverAddr, &config)
	if err != nil {
		console.Error(err)
		return err
	}

	remoteListener, err := sshClientConn.Listen("tcp", remoteAddr)
	if err != nil {
		console.Error(err)
		return err
	}

	for {
		// Setup localConn (type net.Conn)
		remoteConn, err := remoteListener.Accept()
		if err != nil {
			console.Error(err)
			return err
		}

		go func() {

			// Setup localListener (type net.Listener)
			localConn, err := net.Dial("tcp", localAddr)
			if err != nil {
				println(err.Error())
				_ = remoteConn.Close()
				return
			}

			// Copy localConn.Reader to sshConn.Writer
			go func() {
				_, err = io.Copy(localConn, remoteConn)
				if err != nil {
					console.Error(err)
				}
			}()

			// Copy sshConn.Reader to localConn.Writer
			go func() {
				_, err = io.Copy(remoteConn, localConn)
				if err != nil {
					console.Error(err)
				}
			}()

		}()
	}
}

func main() {
	var err = RemoteForward(
		"root", "root", "127.0.0.1:2222",
		"0.0.0.0:80", "127.0.0.1:12356",
	)

	if err != nil {
		// console.Error(err)
	}

	console.Info(1)

	select {}
}
