package backy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

func sshConnect(user, password, host string, port int) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	return ssh.Dial("tcp", addr, config)
}

func sshReadFile(client *ssh.Client, remotePath string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(fmt.Sprintf("cat %s", remotePath)); err != nil {
		return "", err
	}
	return b.String(), nil
}

func sshWriteFile(client *ssh.Client, remotePath, content string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, content)
	}()

	cmd := fmt.Sprintf("cat > %s", remotePath)
	return session.Run(cmd)
}

func lineInString(content, regexpPattern, line string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	found := false
	re := regexp.MustCompile(regexpPattern)

	for scanner.Scan() {
		l := scanner.Text()
		if re.MatchString(l) {
			found = true
			lines = append(lines, line)
		} else {
			lines = append(lines, l)
		}
	}
	if !found {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n") + "\n"
}

func Call() {
	user := "youruser"
	password := "yourpassword"
	host := "yourhost"
	port := 22
	remotePath := "/path/to/remote/file"

	client, err := sshConnect(user, password, host, port)
	if err != nil {
		fmt.Println("SSH connection error:", err)
		return
	}
	defer client.Close()

	content, err := sshReadFile(client, remotePath)
	if err != nil {
		fmt.Println("Read error:", err)
		return
	}

	newContent := lineInString(content, "^foo=", "foo=bar")

	if err := sshWriteFile(client, remotePath, newContent); err != nil {
		fmt.Println("Write error:", err)
		return
	}

	fmt.Println("Line updated successfully over SSH.")
}

type LineInFile struct {
	RemotePath    string         // Path to the remote file
	Pattern       string         // Regex pattern to match lines
	Line          string         // Line to insert or replace
	InsertAfter   bool           // If true, insert after matched line; else replace
	User          string         // SSH username
	Password      string         // SSH password (use key for production)
	Host          string         // SSH host
	Port          int            // SSH port
	regexCompiled *regexp.Regexp // Compiled regex (internal use)
}

// CompileRegex compiles the regex pattern for later use
func (l *LineInFile) CompileRegex() error {
	re, err := regexp.Compile(l.Pattern)
	if err != nil {
		return err
	}
	l.regexCompiled = re
	return nil
}
