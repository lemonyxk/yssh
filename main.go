package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/lemoyxk/console"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {

	home, err := os.UserHomeDir()
	file, err := os.OpenFile(home+"/.yssh/config.json", os.O_RDONLY, 0666)
	if err != nil {
		println("config:", err.Error())
		return
	}

	jsonString, err := ioutil.ReadAll(file)
	if err != nil {
		println("config:", err.Error())
		return
	}

	var configs ServerConfigList

	err = json.Unmarshal(jsonString, &configs)
	if err != nil {
		println("config:", err.Error())
		return
	}

	var table = console.NewTable()
	table.Header("INDEX", "NAME", "HOST")

	for k, v := range configs {
		table.Row(k+1, v.Name, v.Host)
	}

	console.FgGreen.Println(table.Render())

	var config ServerConfig

	for {

		print("Please select server index: ")

		var number int

		if _, err := fmt.Scan(&number); err != nil {
			println("input:", err.Error())
			continue
		}

		if number < 1 || number > len(configs) {
			println("input:", number, "is invalid")
			continue
		}

		config = configs[number-1]

		break

	}

	session, err := connect(config.User, config.Password, config.Host, config.Port)
	if err != nil {
		println("connect:", err.Error())
		return
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		println("terminal:", err)
		return
	}
	defer terminal.Restore(fd, oldState)

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		println("terminal:", err.Error())
		return
	}

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		println("terminal:", err.Error())
		return
	}

	// fmt.Println("Connect to", config.Name, config.Host, "success\n")

	var ticker = time.NewTicker(time.Second * 60)
	go func() {
		for {
			select {
			case <-ticker.C:
				_, err := session.SendRequest(config.User, false, nil)
				if err != nil {
					println("ping:", err.Error())
					os.Exit(0)
				}
			}
		}
	}()

	err = session.Run("$SHELL")
	if err != nil {
		println("exit:", err.Error())
	}

	// fmt.Println("Exit", config.Name, config.Host, "success")
}

func connect(user, password, host string, port int) (*ssh.Session, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		session      *ssh.Session
		err          error
	)

	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		// 需要验证服务端
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// connect to ssh
	addr = fmt.Sprintf("%s:%d", host, port)

	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create session
	if session, err = client.NewSession(); err != nil {
		return nil, err
	}

	return session, nil
}
