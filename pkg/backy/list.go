package backy

import "fmt"

/*
	Command: command [args...]
	Host: Local or remote (list the name)

	List: name
	Commands:
	flags: list commands
	if listcommands: (use list command)
		Command: command [args...]
		Host: Local or remote (list the name)

*/

// ListCommand searches the commands in the file to find one
func (opts *ConfigOpts) ListCommand(cmd string) {
	// bool for commands not found
	// gets set to false if a command is not found
	// set to true if the command is found
	var cmdFound bool = false
	var cmdInfo *Command
	// check commands in file against cmd
	for _, cmdInFile := range opts.executeCmds {
		print(cmdInFile)
		cmdFound = false

		if cmd == cmdInFile {
			cmdFound = true
			cmdInfo = opts.Cmds[cmd]
			break
		}
	}

	// print the command's information
	if cmdFound {

		print("Command: ")

		print(cmdInfo.Cmd)
		if len(cmdInfo.Args) >= 0 {

			for _, v := range cmdInfo.Args {
				print(" ") // print space between command and args
				print(v)   // print command arg
			}
		}

		// is is remote or local
		if cmdInfo.Host != nil {

			print("Host: ", cmdInfo.Host)

		} else {

			print("Host: Runs on Local Machine\n\n")

		}

	} else {

		fmt.Printf("Command %s not found. Check spelling.\n", cmd)

	}

}
