package main

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/lemonyxk/console"
	"github.com/olekukonko/ts"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

//go:embed config.json
var tmp []byte

const SSGTimeout = 3 * time.Second
const SSHRetryTimes = 20
const SSHPingInterval = 30 * time.Second
const SSHProgress = 100 * time.Millisecond

func main() {

	var index = -1

	if len(os.Args) == 1 {
		index = 0
	} else {
		index, _ = strconv.Atoi(os.Args[1])
	}

	home, err := os.UserHomeDir()
	file, err := os.OpenFile(home+"/.yssh/config.json", os.O_RDONLY, 0666)
	if err != nil {
		console.FgRed.Println("config:", err.Error())
		console.FgRed.Println("config: please create config file at ~/.yssh/config.json")
		console.FgRed.Println("config: example:")
		println(string(tmp))
		return
	}

	jsonString, err := io.ReadAll(file)
	if err != nil {
		console.FgRed.Println("config:", err.Error())
		return
	}

	var configList ServerConfigList

	err = json.Unmarshal(jsonString, &configList)
	if err != nil {
		console.FgRed.Println("config:", err.Error())
		return
	}

	if len(configList) == 0 {
		console.FgRed.Println("config: no config")
		return
	}

	if index > len(configList)-1 {
		console.FgRed.Println("config: index overflow")
		return
	}

	var configs = configList[index]

	var table = console.NewTable()
	table.Header("INDEX", "NAME", "HOST")

	for k, v := range configs {
		table.Row(k+1, v.Name, v.Host)
	}

	var config ServerConfig

	for {
		console.FgYellow.Println(table.Render())

		var number int

		for {

			console.FgCyan.Printf("Please select server index: ")

			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			var text = scanner.Text()
			number, err = strconv.Atoi(text)
			if err != nil {
				console.FgRed.Println("input:", text, "is not a number")
				continue
			}

			if number < 1 || number > len(configs) {
				console.FgRed.Println("input:", number, "is overflow index")
				continue
			}

			break
		}

		config = configs[number-1]

		do(config)

		break
	}

	// println("Exit", config.Name, config.Host, "success\r")
}

func do(config ServerConfig) {

	var session *ssh.Session
	var err error

	var now = time.Now()
	var timeoutTicker = time.NewTicker(SSHProgress)
	var stopTimeoutTicker = make(chan struct{})
	go func() {
		for {
			select {
			case <-timeoutTicker.C:
				console.FgGreen.Printf(
					"\rStart connecting to %s %s %.1fs",
					config.Name, config.Host,
					float64(time.Now().Sub(now).Milliseconds())/1000,
				)
			case <-stopTimeoutTicker:
				return
			}
		}
	}()

	var sshRetryTimes = 0

	for {
		session, err = NewSession(Config{
			UserName:   config.User,
			Password:   config.Password,
			Addr:       fmt.Sprintf("%s:%d", config.Host, config.Port),
			PrivateKey: config.PrivateKey,
			Timeout:    SSGTimeout,
		})
		if err != nil {
			sshRetryTimes++
			if sshRetryTimes == SSHRetryTimes {
				console.FgRed.Println("\nconnect:", err.Error())
				return
			}
			continue
		}

		timeoutTicker.Stop()
		stopTimeoutTicker <- struct{}{}
		break
	}

	defer func() { _ = session.Close() }()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		console.FgRed.Println("\nterminal: make raw", err)
		return
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	size, err := ts.GetSize()
	if err != nil {
		console.FgRed.Println("\nterminal: get size", err.Error())
		return
	}

	termWidth, termHeight := size.Col(), size.Row()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		console.FgRed.Println("\nterminal: request pty", err.Error())
		return
	}

	console.FgGreen.Println("\r\nConnect to", config.Name, config.Host, "success\r")

	var pingTimeoutTicker = time.NewTicker(SSHPingInterval)
	var stopPingTimeoutTicker = make(chan struct{})
	go func() {
		for {
			select {
			case <-pingTimeoutTicker.C:
				_, err := session.SendRequest(config.User, false, nil)
				if err != nil {
					console.FgRed.Println("ping:", err.Error())
					_ = session.Close()
					return
				}
			case <-stopPingTimeoutTicker:
				return
			}
		}
	}()

	_ = session.Run("$SHELL")

	stopPingTimeoutTicker <- struct{}{}
}
