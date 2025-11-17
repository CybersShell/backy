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
	for cmdInFile := range opts.Cmds {
		cmdFound = false

		if cmd == cmdInFile {
			cmdFound = true
			cmdInfo = opts.Cmds[cmd]
			break
		}
	}

	// print the command's information
	if cmdFound {

		println("Command: ")

		print(cmdInfo.Cmd)

		for _, v := range cmdInfo.Args {
			print(" ") // print space between command and args
			print(v)   // print command arg
		}

		// is it remote or local
		if !IsHostLocal(cmdInfo.Host) {
			println()
			print("Host: ", cmdInfo.Host)
			println()

		} else {

			println()
			print("Host: Runs on Local Machine\n\n")

		}

		if cmdInfo.Dir != nil {
			println()
			print("Directory: ", *cmdInfo.Dir)
			println()
		}

		if cmdInfo.Type.String() != "" {
			print("Type: ", cmdInfo.Type.String())
			println()
		}

	} else {

		fmt.Printf("Command %s not found. Check spelling.\n", cmd)

	}

}

func (opts *ConfigOpts) ListCommandList(list string) {
	// bool for commands not found
	// gets set to false if a command is not found
	// set to true if the command is found
	var listFound bool
	var listInfo *CmdList
	// check commands in file against cmd
	for listInFile, l := range opts.CmdConfigLists {
		listFound = false

		if list == listInFile {
			listFound = true
			listInfo = l
			break
		}
	}

	// print the command's information
	if listFound {

		println("List: ", list)
		println()

		for _, v := range listInfo.Order {
			println()
			opts.ListCommand(v)
		}

	} else {

		fmt.Printf("List %s not found. Check spelling.\n", list)

	}
}
