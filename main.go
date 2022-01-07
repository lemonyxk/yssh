package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/lemoyxk/console"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

const SSGTimeout = 3 * time.Second
const SSHRetryTimes = 20
const SSHPingInterval = 30 * time.Second
const SSHProgress = 100 * time.Millisecond

func main() {

	home, err := os.UserHomeDir()
	file, err := os.OpenFile(home+"/.yssh/config.json", os.O_RDONLY, 0666)
	if err != nil {
		console.FgRed.Println("config:", err.Error())
		return
	}

	jsonString, err := ioutil.ReadAll(file)
	if err != nil {
		console.FgRed.Println("config:", err.Error())
		return
	}

	var configs ServerConfigList

	err = json.Unmarshal(jsonString, &configs)
	if err != nil {
		console.FgRed.Println("config:", err.Error())
		return
	}

	var table = console.NewTable()
	table.Header("INDEX", "NAME", "HOST")

	for k, v := range configs {
		table.Row(k+1, v.Name, v.Host)
	}

	console.FgYellow.Println(table.Render())

	var config ServerConfig

	for {

		console.FgCyan.Printf("Please select server index: ")

		var number int

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

		config = configs[number-1]

		break

	}

	var session *ssh.Session

	var now = time.Now()
	var timeoutTicker = time.NewTicker(SSHProgress)
	go func() {
		for {
			select {
			case <-timeoutTicker.C:
				console.FgGreen.Printf(
					"\rStart connecting to %s %s %.1fs",
					config.Name, config.Host,
					float64(time.Now().Sub(now).Milliseconds())/1000,
				)
			}
		}
	}()

	var sshRetryTimes = 0

	for {
		session, err = connect(config.User, config.Password, config.Host, config.Port)
		if err != nil {
			sshRetryTimes++
			if sshRetryTimes == SSHRetryTimes {
				console.FgRed.Println("\nconnect:", err.Error())
				return
			}
			continue
		}
		timeoutTicker.Stop()
		break
	}

	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		console.FgRed.Println("terminal:", err)
		return
	}
	defer terminal.Restore(fd, oldState)

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		console.FgRed.Println("terminal:", err.Error())
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
		console.FgRed.Println("terminal:", err.Error())
		return
	}

	console.FgGreen.Println("\r\nConnect to", config.Name, config.Host, "success\r")

	var ticker = time.NewTicker(SSHPingInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				_, err := session.SendRequest(config.User, false, nil)
				if err != nil {
					console.FgRed.Println("ping:", err.Error())
					os.Exit(0)
				}
			}
		}
	}()

	_ = session.Run("$SHELL")

	// println("Exit", config.Name, config.Host, "success\r")
}
