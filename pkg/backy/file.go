package backy

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type LocalFileCommandExecutor struct{}

func (f *LocalFileCommandExecutor) Execute(cmd *Command) error {

	localExecutor := LocalCommandExecutor{}

	switch cmd.FileOperation {
	case "copy":
		return localExecutor.copyFile(cmd.Source, cmd.Destination, cmd.Permissions)
	default:
		return fmt.Errorf("unsupported file operation: %s", cmd.FileOperation)
	}
}

func (f *LocalFileCommandExecutor) ReadLocalFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (f *LocalFileCommandExecutor) WriteLocalFile(path string, data []byte, Perms fs.FileMode) error {
	err := os.WriteFile(path, data, Perms)
	if err != nil {
		return err
	}
	return nil
}

func (f *LocalCommandExecutor) copyFile(source, destination string, Perms fs.FileMode) error {
	input, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	err = os.WriteFile(destination, input, Perms)
	if err != nil {
		return err
	}
	return nil
}

type RemoteFileCommandExecutor struct{}

func (r *RemoteFileCommandExecutor) Execute(cmd *Command) error {

	remoteExecutor := RemoteFileCommandExecutor{}
	sourceTypeLocal := false

	if cmd.SourceType == "local" {
		sourceTypeLocal = true
	}
	switch cmd.FileOperation {
	case "copy":
		return remoteExecutor.copyFile(cmd.Source, cmd.Destination, cmd.Permissions, cmd.RemoteHost.SshClient, sourceTypeLocal)
	default:
		return fmt.Errorf("unsupported file operation: %s", cmd.FileOperation)
	}
}

func (r *RemoteFileCommandExecutor) copyFile(source, destination string, Perms fs.FileMode, sshClient *ssh.Client, sourceTypeLocal bool) error {
	if sourceTypeLocal {
		input, err := os.ReadFile(source)
		if err != nil {
			return err
		}

		sftpClient, sftpErr := sftp.NewClient(sshClient)
		if sftpErr != nil {
			return sftpErr
		}
		defer sftpClient.Close()

		destFile, err := sftpClient.Create(destination)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = destFile.Write(input)
		if err != nil {
			return err
		}

		err = destFile.Chmod(Perms)
		if err != nil {
			return err
		}

		return nil
	}

	client, err := sftp.NewClient(sshClient)

	if err != nil {
		return err
	}

	srcFile, err := client.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := client.Create(destination)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = srcFile.WriteTo(destFile)
	if err != nil {
		return err
	}

	err = destFile.Chmod(Perms)
	if err != nil {
		return err
	}

	return nil
}

func (r *RemoteFileCommandExecutor) ReadRemoteFile(path string, sshClient *ssh.Client) ([]byte, error) {
	sftpClient, sftpErr := sftp.NewClient(sshClient)
	if sftpErr != nil {
		return nil, sftpErr
	}
	defer sftpClient.Close()

	file, err := sftpClient.Open(path)

	if err != nil {
		return nil, err
	}
	defer file.Close()
	var fileData []byte

	_, err = file.Read(fileData)
	if err != nil {
		return nil, err
	}
	return fileData, nil
}

func (r *RemoteFileCommandExecutor) WriteRemoteFile(path string, data []byte, Perms fs.FileMode, sshClient *ssh.Client) error {
	sftpClient, sftpErr := sftp.NewClient(sshClient)
	if sftpErr != nil {
		return sftpErr
	}
	defer sftpClient.Close()

	file, err := sftpClient.Create(path)

	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	err = file.Chmod(Perms)
	if err != nil {
		return err
	}

	return nil
}

type FileCommandExecutor interface {
	Execute(cmd *Command) error
}

func NewFileCommandExecutor(isRemote bool) FileCommandExecutor {
	if isRemote {
		return &RemoteFileCommandExecutor{}
	}
	return &LocalFileCommandExecutor{}
}
