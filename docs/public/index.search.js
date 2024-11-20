var relearn_search_index = [
  {
    "content": "The yaml top-level map can be any string.\nThe top-level name must be unique.\nExample Config commands: stop-docker-container: cmd: docker Args: - compose - -f /some/path/to/docker-compose.yaml - down # if host is not defined, command will be run locally # The host has to be defined in either the config file or the SSH Config files host: some-host hooks error: - some-other-command-when-failing success: - success-command final: - final-command backup-docker-container-script: cmd: /path/to/local/script # script file is input as stdin to SSH type: scriptFile # also can be script environment: - FOO=BAR - APP=$VAR Values available for this section:\nname notes type required cmd Defines the command to execute string yes args Defines the arguments to the command []string no environment Defines evironment variables for the command []string no type May be scriptFile or script. Runs script from local machine on remote. Only applicable when host is defined. string no getOutput Command(s) output is in the notification(s) bool no host If not specified, the command will execute locally. string no scriptEnvFile When type is scriptFile, the script is appended to this file. string no shell Only applicable when host is not specified string no hooks Hooks are used at the end of the individual command. Must be another command. []string no cmd cmd must be a valid command or script to execute.\nargs args must be arguments to cmd as they would be on the command-line:\ncmd [arg1 arg2 ...] Define them in an array:\nargs: - arg1 - arg2 - arg3 getOutput Get command output when a notification is sent.\nIs not required. Can be true or false.\nhost Info If any host is not defined or left blank, the command will run on the local machine.\nHost may or may not be defined in the hosts section.\nInfo If any host from the commands section does not match any object in the hosts section, the Host is assumed to be this value. This value will be used to search in the default SSH config files.\nFor example, say that I have a host defined in my SSH config with the Host defined as web-prod. If I assign a value to host as host: web-prod and don’t specify this value in the hosts object, web-prod will be used as the Host in searching the SSH config files.\nshell If shell is defined and host is NOT defined, the command will run in the specified shell. Make sure to escape any shell input.\nscriptEnvFile Path to a file.\nWhen type is specified, the script is appended to this file.\nThis is useful for specifying environment variables or other things so they don’t have to be included in the script.\ntype May be scriptFile or script. Runs script from local machine on remote host passed to the SSH session as standard input.\nIf type is script, cmd is used as the script.\nIf type is scriptFile, cmd must be a script file.\nenvironment The environment variables support expansion:\nusing escaped values $VAR or ${VAR} For now, the variables have to be defined in an .env file in the same directory as the config file.\nIf using it with host specified, the SSH server has to be configured to accept those env variables.\nIf the command is run locally, the OS’s environment is added.\nhooks Hooks are run after the command is run.\nErrors are run if the command errors, success if it returns no error. Final hooks are run regardless of error condition.\nValues for hooks are as follows:\ncommand: hook: # these commands are defined elsewhere in the file error: - errcommand success: - successcommand final: - donecommand ",
    "description": "",
    "tags": null,
    "title": "Commands",
    "uri": "/config/commands/index.html"
  },
  {
    "content": "Binaries are available from the release page. Make sure to get the correct version for your system, which supports x86_64, ARM64, and i386.\nSource Install You can install from source. You will need Go installed.\nThen run:\ngo install git.andrewnw.xyz/CyberShell/backy@master Once set, jump over to the config docs and start configuring your file.\n",
    "description": "This page tells you how to install Backy.\n",
    "tags": null,
    "title": "Install Backy",
    "uri": "/getting-started/install/index.html"
  },
  {
    "content": "Command lists are for executing commands in sequence and getting notifications from them.\nThe top-level object key can be anything you want but not the same as another.\nLists can go in a separate file. Command lists should be in a separate file if:\nkey ‘cmd-lists.file’ is found hosts.yml or hosts.yaml is found in the same directory as the backy config file test2: name: test2 order: - test - test2 notifications: - mail.prod-email - matrix.sysadmin cron: \"0 * * * * *\" key description type required order Defines the sequence of commands to execute []string yes getOutput Command(s) output is in the notification(s) bool no notifications The notification service(s) and ID(s) to use on success and failure. Must be service.id. See the notifications documentation page for more []string no name Optional name of the list string no cron Time at which to schedule the list. Only has affect when cron subcommand is run. string no Order The order is an array of commands to execute in order. Each command must be defined.\norder: - cmd-1 - cmd-2 getOutput Get command output when a notification is sent.\nIs not required. Can be true or false. Default is false.\nNotifications An array of notification IDs to use on success and failure. Must match any of the notifications object map keys.\nName Name is optional. If name is not defined, name will be the object’s map key.\nCron mode Backy also has a cron mode, so one can run backy cron and start a process that schedules jobs to run at times defined in the configuration file.\nAdding cron: 0 0 1 * * * to a cmd-lists object will schedule the list at 1 in the morning. See https://crontab.guru/ for reference.\nTip Note: Backy uses the second field of cron, so add anything except * to the beginning of a regular cron expression.\ncmd-lists: docker-container-backup: # this can be any name you want # all commands have to be defined order: - stop-docker-container - backup-docker-container-script - shell-cmd - hostname - start-docker-container notifications: - matrix.id name: backup-some-container cron: \"0 0 1 * * *\" hostname: name: hostname order: - hostname notifications: - mail.prod-email ",
    "description": "This page tells you how to get started with Backy.\n",
    "tags": null,
    "title": "Command Lists",
    "uri": "/config/command-lists/index.html"
  },
  {
    "content": "If you have not installed Backy, see the install documentation.\nIf you need to configure it, see the config page.\n",
    "description": "This page tells you how to get started with Backy.\n",
    "tags": null,
    "title": "Getting started",
    "uri": "/getting-started/index.html"
  },
  {
    "content": "This is the section on the config file.\nTo use a specific file: backy [command] -f /path/to/file\nIf you leave the config path blank, the following paths will be searched in order:\n./backy.yml ./backy.yaml ~/.config/backy.yml ~/.config/backy.yaml Create a file at ~/.config/backy.yml.\nSee the rest of the documentation in this section to configure it.\n",
    "description": "This page tells you how to configure Backy.\n",
    "tags": null,
    "title": "Configuring Backy",
    "uri": "/config/index.html"
  },
  {
    "content": "Notifications can be sent on command list completion and failure.\nThe supported platforms for notifications are email (SMTP) and Matrix.\nNotifications are defined by service, with the current form following below. Ids must come after the service.\nnotifications: mail: prod-email: host: yourhost.tld port: 587 senderaddress: email@domain.tld to: - admin@domain.tld username: smtp-username@domain.tld password: your-password-here matrix: matrix: home-server: your-home-server.tld room-id: room-id access-token: your-access-token user-id: your-user-id Sections recognized are mail and matrix\nThere must be a section with an id (eg. mail.test-svr) following one of these sections.\nmail key description type host Specifies the SMTP host to connect to string port Specifies the SMTP port uint16 senderaddress Address from which to send mail string to Recipients to send emails to []string username SMTP username string password SMTP password string matrix key description type home-server Specifies the Matrix server connect to string room-id Specifies the room ID of the room to send messages to string access-token Matrix access token string user-id Matrix user ID string To get your access token (assumes you are using Element) :\nLog in to the account you want to get the access token for. Click on the name in the top left corner, then “Settings”. Click the “Help \u0026 About” tab (left side of the dialog). Scroll to the bottom and click on \u003cclick to reveal\u003e part of Access Token. Copy your access token to a safe place. To get the room ID:\nOn Element or a similar client, navigate to the room. Navigate to the settings from the top menu. Click on Advanced, the room ID is there. Info Make sure to quote the room ID, as YAML spec defines tags using !.\n",
    "description": "This page tells you how to get set up Backy notifications.\n",
    "tags": null,
    "title": "Notifications",
    "uri": "/config/notifications/index.html"
  },
  {
    "content": "This page lists documentation for the CLI.\nBacky Backy is a command-line application useful for configuring backups, or any commands run in sequence. Usage: backy [command] Available Commands: backup Runs commands defined in config file. completion Generate the autocompletion script for the specified shell cron Starts a scheduler that runs lists defined in config file. exec Runs commands defined in config file in order given. help Help about any command list Lists commands, lists, or hosts defined in config file. version Prints the version and exits Flags: -f, --config string config file to read from -h, --help help for backy -v, --verbose Sets verbose level Use \"backy [command] --help\" for more information about a command. Subcommands backup Backup executes commands defined in config file. Use the --lists or -l flag to execute the specified lists. If not flag is not given, all lists will be executed. Usage: backy backup [--lists=list1,list2,... | -l list1, list2,...] [flags] Flags: -h, --help help for backup -l, --lists strings Accepts comma-separated names of command lists to execute. Global Flags: -f, --config string config file to read from -v, --verbose Sets verbose level cron Cron starts a scheduler that executes command lists at the time defined in config file. Usage: backy cron [flags] Flags: -h, --help help for cron Global Flags: -f, --config string config file to read from -v, --verbose Sets verbose level exec Exec executes commands defined in config file in order given. Usage: backy exec command ... [flags] Flags: -h, --help help for exec Global Flags: -f, --config string config file to read from -v, --verbose Sets verbose level version Prints the version and exits. No arguments just prints the version number only. Usage: backy version [flags] Flags: -h, --help help for version -n, --num Output the version number only. -V, --vpre Output the version with v prefixed. Global Flags: -f, --config string config file to read from -v, --verbose Sets verbose level list Backup lists commands or groups defined in config file. Use the --lists or -l flag to list the specified lists. If not flag is not given, all lists will be executed. Usage: backy list [--list=list1,list2,... | -l list1, list2,...] [ -cmd cmd1 cmd2 cmd3...] [flags] Flags: -c, --cmds strings Accepts comma-separated names of commands to list. -h, --help help for list -l, --lists strings Accepts comma-separated names of command lists to list. Global Flags: -f, --config string config file to read from -v, --verbose Sets verbose level ",
    "description": "",
    "tags": null,
    "title": "CLI",
    "uri": "/cli/index.html"
  },
  {
    "content": "Vault is a tool for storing secrets and other data securely.\nVault config can be used by prefixing vault: in front of a password or ENV var.\nThis is the object in the config file:\nvault: token: hvs.tXqcASvTP8wg92f7riyvGyuf address: http://127.0.0.1:8200 enabled: false keys: - name: mongourl mountpath: secret path: mongo/url type: # KVv1 or KVv2 - name: path: type: mountpath: ",
    "description": "",
    "tags": null,
    "title": "Vault",
    "uri": "/config/vault/index.html"
  },
  {
    "content": "The repo mirrors are:\nhttps://git.andrewnw.xyz/CyberShell/backy https://github.com/CybersShell/backy ",
    "description": "",
    "tags": null,
    "title": "Repositories",
    "uri": "/repositories/index.html"
  },
  {
    "content": "Backy is a tool for automating data backup and remote command execution. It can work over SSH, and provides completion and failure notifications, error reporting, and more.\nWhy the name Backy? Because I wanted an app for backups.\nView the changelog here.\nTip Feel free to open a PR, raise an issue(s), or request new feature(s).\nFeatures Allows easy configuration of executable commands\nAllows for commands to be run on many hosts over SSH\nCommands can be grouped in list to run in specific order\nNotifications on completion and failure\nRun in cron mode\nFor any command, especially backup commands\n",
    "description": "",
    "tags": null,
    "title": "Backy",
    "uri": "/index.html"
  },
  {
    "content": "",
    "description": "",
    "tags": null,
    "title": "Categories",
    "uri": "/categories/index.html"
  },
  {
    "content": "Commands The commands section is for defining commands. These can be run with or without a shell and on a host or locally.\nSee the commands documentation for further information.\ncommands: stop-docker-container: output: true # Optional and only when run in list and notifications are sent cmd: docker args: - compose - -f /some/path/to/docker-compose.yaml - down # if host is not defined, cmd will be run locally host: some-host backup-docker-container-script: cmd: /path/to/script # The host has to be defined in the config file host: some-host environment: - FOO=BAR - APP=$VAR # defined in .env file in config directory shell-cmd: cmd: rsync shell: bash args: - -av - some-host:/path/to/data - ~/Docker/Backups/docker-data script: type: scriptFile # run a local script on a remote host cmd: path/to/your/script.sh host: some-host hostname: cmd: hostname Lists To execute groups of commands in sequence, use a list configuration.\ncmd-lists: cmds-to-run: # this can be any name you want # all commands have to be defined in the commands section order: - stop-docker-container - backup-docker-container-script - shell-cmd - hostname getOutput: true # Optional and only for when notifications are sent notifications: - matrix name: backup-some-server hostname: name: hostname order: - hostname notifications: - prod-email Hosts The hosts object may or may not be defined.\nInfo If any host from a commands object does not match any host object, the needed values will be checked in the default SSH config files.\nhosts: # any needed ssh_config(5) keys/values not listed here will be looked up in the config file or the default config file some-host: hostname: some-hostname config: ~/.ssh/config user: user privatekeypath: /path/to/private/key port: 22 # can also be env:VAR or the password itself password: file:/path/to/file # can also be env:VAR or the password itself privatekeypassword: file:/path/to/file # only one is supported for now proxyjump: some-proxy-host Notifications The notifications object can have two forms.\nFor more, see the notification object documentation. The top-level map key is id that has to be referenced by the cmd-lists key notifications.\nnotifications: prod-email: type: mail host: yourhost.tld port: 587 senderAddress: email@domain.tld recipients: - admin@domain.tld username: smtp-username@domain.tld password: your-password-here matrix: type: matrix home-server: your-home-server.tld room-id: room-id access-token: your-access-token user-id: your-user-id Logging cmd-std-out controls whether commands output is echoed to StdOut.\nIf logfile is not defined, the log file will be written to the config directory in the file backy.log.\nconsole-disabled controls whether the logging messages are echoed to StdOut. Default is false.\nverbose basically does nothing as all necessary info is already output.\nlogging: verbose: false file: path/to/log/file.log console-disabled: false cmd-std-out: false Vault Vault can be used to get some configuration values and ENV variables securely.\nvault: token: hvs.tXqcASvTP8wg92f7riyvGyuf address: http://127.0.0.1:8200 enabled: false keys: - name: mongourl mountpath: secret path: mongo/url type: # KVv1 or KVv2 - name: path: type: mountpath: ",
    "description": "This page tells you how to configure Backy.\n",
    "tags": null,
    "title": "Config File Definitions",
    "uri": "/getting-started/config/index.html"
  },
  {
    "content": "",
    "description": "",
    "tags": null,
    "title": "Tags",
    "uri": "/tags/index.html"
  }
]
