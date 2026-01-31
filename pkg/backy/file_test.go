package backy

import (
	"os"
	"testing"
)

func TestCopyFileLocal(t *testing.T) {
	FileCommand := Command{}
	FileCommand.Type = FileCommandType
	FileCommand.FileOperation = "copy"
	FileCommand.Permissions = 0644
	FileCommand.Source = "/home/andrew/Projects/backy/tests/data/fileops/source.txt"
	FileCommand.Destination = "/home/andrew/Projects/backy/tests/data/fileops/destination.txt"
	var FileCommandExecutor = LocalFileCommandExecutor{}
	err := FileCommandExecutor.Execute(&FileCommand)
	if err != nil {
		t.Errorf("Error executing file command: %v", err)
	}

	srcBytes, srcErr := os.ReadFile(FileCommand.Source)
	if srcErr != nil {
		t.Errorf("Error reading source file: %v", srcErr)
	}
	destBytes, destErr := os.ReadFile(FileCommand.Destination)
	if destErr != nil {
		t.Errorf("Error reading destination file: %v", destErr)
	}
	if string(srcBytes) != string(destBytes) {
		t.Errorf("Source and destination files do not match")
	}

	// Additional checks can be added here to verify the file was copied correctly
}

func TestCopyFileRemote(t *testing.T) {
	opts := NewConfigOptions("")
	opts.Hosts = map[string]*Host{}
	RemoteFileCommand := Command{}
	RemoteFileCommand.Type = FileCommandType
	RemoteFileCommand.FileOperation = "copy"
	RemoteFileCommand.Permissions = 0644
	RemoteFileCommand.Destination = "/home/backy/destination.txt"
	RemoteFileCommand.Source = "/home/andrew/Projects/backy/tests/data/fileops/source.txt"
	RemoteFileCommand.Host = "localhost"
	RemoteFileCommand.RemoteHost = &Host{
		HostName:       "localhost",
		User:           "backy",
		Password:       "backy",
		Port:           2222,
		PrivateKeyPath: "/home/andrew/Projects/backy/tests/docker/backytest",
		KnownHostsFile: "/home/andrew/Projects/backy/tests/docker/known_hosts",
	}

	sshErr := RemoteFileCommand.RemoteHost.ConnectToHost(opts)
	if sshErr != nil {
		t.Errorf("Error connecting to remote host: %v", sshErr)
	}
	var RemoteFileCommandExecutor = RemoteFileCommandExecutor{}
	err := RemoteFileCommandExecutor.Execute(&RemoteFileCommand)
	if err != nil {
		t.Errorf("Error executing remote file command: %v", err)
	}

	srcBytes, srcErr := os.ReadFile(RemoteFileCommand.Source)
	if srcErr != nil {
		t.Errorf("Error reading source file: %v", srcErr)
	}
	destBytes, destErr := RemoteFileCommandExecutor.ReadRemoteFile(RemoteFileCommand.Destination, RemoteFileCommand.RemoteHost.SshClient)
	if destErr != nil {
		t.Errorf("Error reading destination file: %v", destErr)
	}
	if string(srcBytes) != string(destBytes) {
		t.Errorf("Source and destination files do not match")
	}

	// Additional checks can be added here to verify the file was copied correctly
}
