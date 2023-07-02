var relearn_search_index = [
  {
    "content": "Binaries are available from the release page. Make sure to get the correct version for your system, which supports x86_64, ARM64, and i386.\nSource Install You can install from source. You will need Go installed.\nThen run:\ngo install git.andrewnw.xyz/CyberShell/backy@master Once set, jump over to the config docs and start configuring your file.\n",
    "description": "This page tells you how to install Backy.\n",
    "tags": null,
    "title": "Install Backy",
    "uri": "/getting-started/install/index.html"
  },
  {
    "content": "Command lists are for executing commands in sequence and getting notifications from them.\nThe top-level object key can be anything you want.\nkey description type required order Defines the sequence of commands to execute []string yes notifications The notification IDs to use on success and failure []string no name Optional name of the list string no cron Time at which to schedule the list. string no Order The order is an array of commands to execute in order. Each command must be defined.\nNotifications An array of notification IDs to use on success and failure. Must match any of the notifications object map keys.\nName Name is optional for logging. If name is not defined, name will be the object’s map key.\nCron mode Backy also has a cron mode, so one can run backy cron and start a process that schedules jobs to run at times defined in the configuration file.\nAdding cron: 0 0 1 * * * to a cmd-configs object will schedule the list at 1 in the morning. See https://crontab.guru/ for reference.\nTip Note: Backy uses the second field of cron, so add anything except * to the beginning of a regular cron expression.\ncmd-configs: cmds-to-run: # this can be any name you want # all commands have to be defined order: - stop-docker-container - backup-docker-container-script - shell-cmd - hostname notifications: - matrix name: backup-some-server cron: \"0 0 1 * * *\" hostname: name: hostname order: - hostname notifications: - prod-email ",
    "description": "This page tells you how to get started with Backy.\n",
    "tags": null,
    "title": "Command Lists",
    "uri": "/config/command-lists/index.html"
  },
  {
    "content": "The yaml top-level map can be any string.\nThe top-level name must be unique.\ncommands: stop-docker-container: cmd: docker Args: - compose - -f /some/path/to/docker-compose.yaml - down # if host is not defined, command will be run locally host: some-host backup-docker-container-script: cmd: /path/to/script # The host has to be defined in either the config file or the SSH Config files host: some-host environment: - FOO=BAR - APP=$VAR Values available for this section:\nname description type required cmd Defines the command to execute string yes args Defines the arguments to the command []string no environment Defines evironment variables for the command []string no host If not specified, the command will execute locally. string no shell Only applicable when host is not specified string no cmd cmd must be a valid command or script to execute.\nargs args must be arguments to cmd as they would be on the command-line:\ncmd [arg1 arg2 ...] Define them in an array:\nargs: - arg1 - arg2 - arg3 host Info If any host is not defined or left blank, the command will run on the local machine.\nHost may or may not be defined in the hosts section.\nInfo If any host from the commands section does not match any object in the hosts section, the Host is assumed to be this value. This value will be used to search in the default SSH config files.\nFor example, say that I have a host defined in my SSH config with the Host defined as web-prod. If I assign a value to host as host: web-prod and don’t specify this value in the hosts object, web-prod will be used as the Host in searching the SSH config files.\nshell If shell is defined and host is NOT defined, the command will run in the specified shell. Make sure to escape any shell input.\nenvironment The environment variables support expansion:\nusing escaped values $VAR or ${VAR} For now the variables have to be defined in an .env file in the same directory as the config file.\nIf using it with host specified, the SSH server has to be configured to accept those env variables.\nIf the command is run locally, the OS’s environment is added.\n",
    "description": "",
    "tags": null,
    "title": "Commands",
    "uri": "/config/commands/index.html"
  },
  {
    "content": "If you have not installed Backy, see the install documentation.\n",
    "description": "This page tells you how to get started with Backy.\n",
    "tags": null,
    "title": "Getting started",
    "uri": "/getting-started/index.html"
  },
  {
    "content": "Notifications can be sent on command list completion and failure.\nThe supported platforms for notifications are email (SMTP) and Matrix.\nNotifications are defined by type. The top-level object will be the id, and the type is required.\nInfo Type in a cmd-configs object must match one of these.\nnotifications: prod-email: type: mail host: yourhost.tld port: 587 senderaddress: email@domain.tld to: - admin@domain.tld username: smtp-username@domain.tld password: your-password-here matrix: type: matrix home-server: your-home-server.tld room-id: room-id access-token: your-access-token user-id: your-user-id Types recognized are type: mail and type: matrix\nThe type’s object and its keys are listed below.\ntype: mail key description type host Specifies the SMTP host to connect to string port Specifies the SMTP port uint16 senderaddress Address from which to send mail string to Recipients to send emails to []string username SMTP username string password SMTP password string type: matrix key description type home-server Specifies the Matrix server connect to string room-id Specifies the room ID of the room to send messages to string access-token Matrix access token string user-id Matrix user ID string To get your access token (assumes you are using Element) :\nLog in to the account you want to get the access token for. Click on the name in the top left corner, then “Settings”. Click the “Help \u0026 About” tab (left side of the dialog). Scroll to the bottom and click on \u003cclick to reveal\u003e part of Access Token. Copy your access token to a safe place. To get the room ID:\nOn Element or a similar client, navigate to the room. Navigate to the settings from the top menu. Click on Advanced, the room ID is there. Info Make sure to quote the room ID, as YAML spec defines tags using !.\n",
    "description": "This page tells you how to get set up Backy notifications.\n",
    "tags": null,
    "title": "Notifications",
    "uri": "/config/notifications/index.html"
  },
  {
    "content": "This is the section on the config file.\nTo use a specific file: backy backup -f /path/to/file\nIf you leave the config path blank, the following paths will be searched in order:\n./backy.yml ./backy.yaml ~/.config/backy.yml ~/.config/backy.yaml Create a file at ~/.config/backy.yml.\nSee the documentation in this section to configure it.\n",
    "description": "This page tells you how to configure Backy.\n",
    "tags": null,
    "title": "Configuring Backy",
    "uri": "/config/index.html"
  },
  {
    "content": "This page lists documentation for the CLI.\nSubcommands backy backup Backup executes commands defined in config file. Use the --lists flag to execute the specified commands. If not specified, all lists will be executed. Usage: backy backup [--lists=list1,list2] [flags] Flags: -h, --help help for backup -l, --lists strings Accepts comma-separated names of command lists to execute. Global Flags: -f, --config string config file to read from -v, --verbose Sets verbose level backy exec Exec executes commands defined in config file. Usage: backy exec command ... [flags] Flags: -h, --help help for exec Global Flags: -f, --config string config file to read from -v, --verbose Sets verbose level backy cron Cron starts a scheduler that executes command lists at the time defined in config file. Usage: backy cron [flags] Flags: -h, --help help for cron Global Flags: -f, --config string config file to read from -v, --verbose Sets verbose level backy version Prints the version and exits. Usage: backy version [flags] Flags: -h, --help help for version ",
    "description": "",
    "tags": null,
    "title": "CLI",
    "uri": "/cli/index.html"
  },
  {
    "content": "The repo mirrors are:\nhttps://git.andrewnw.xyz/CyberShell/backy https://git.vern.cc/cybershell/backy https://github.com/CybersShell/backy ",
    "description": "",
    "tags": null,
    "title": "Repositories",
    "uri": "/repositories/index.html"
  },
  {
    "content": "Backy is a tool for automating data backup and remote command execution. It can work over SSH, and provides completion and failure notifications, error reporting, and more.\nWhy the name Backy? Because I wanted an app for backups.\nTip Feel free to open a PR, raise an issue(s), or request new feature(s).\nFeatures Allows easy configuration of executable commands\nAllows for commands to be run on many hosts over SSH\nCommands can be grouped in list to run in specific order\nNotifications on completion and failure\nRun in cron mode\nFor any command, especially backup commands\n",
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
    "content": "Commands The commands section is for defining commands. These can be run with or without a shell and on a host or locally.\nSee the commands documentation for further information.\ncommands: stop-docker-container: cmd: docker args: - compose - -f /some/path/to/docker-compose.yaml - down # if host is not defined, cmd will be run locally host: some-host backup-docker-container-script: cmd: /path/to/script # The host has to be defined in the config file host: some-host environment: - FOO=BAR - APP=$VAR # defined in .env file in config directory shell-cmd: cmd: rsync shell: bash args: - -av - some-host:/path/to/data - ~/Docker/Backups/docker-data hostname: cmd: hostname Lists To execute groups of commands in sequence, use a list configuration.\ncmd-configs: cmds-to-run: # this can be any name you want # all commands have to be defined order: - stop-docker-container - backup-docker-container-script - shell-cmd - hostname notifications: - matrix name: backup-some-server hostname: name: hostname order: - hostname notifications: - prod-email Hosts The hosts object may or may not be defined.\nInfo If any host from a commands object does not match any host object, the needed values will be checked in the default SSH config files.\nhosts: # any ssh_config(5) keys/values not listed here will be looked up in the config file or the default config file some-host: hostname: some-hostname config: ~/.ssh/config user: user privatekeypath: /path/to/private/key port: 22 # can also be env:VAR or the password itself password: file:/path/to/file # can also be env:VAR or the password itself privatekeypassword: file:/path/to/file # only one is supported for now proxyjump: some-proxy-host Notifications The notifications object can have two forms.\nFor more, see the notification object documentation. The top-level map key is id that has to be referenced by the cmd-configs key notifications.\nnotifications: prod-email: type: mail host: yourhost.tld port: 587 senderAddress: email@domain.tld recipients: - admin@domain.tld username: smtp-username@domain.tld password: your-password-here matrix: type: matrix home-server: your-home-server.tld room-id: room-id access-token: your-access-token user-id: your-user-id Logging cmd-std-out controls whether commands output is echoed to StdOut.\nIf logfile is not defined, the log file will be written to the config directory in the file backy.log.\nconsole-disabled controls whether the logging messages are echoed to StdOut. Default is false.\nverbose basically does nothing as all necessary info is already output.\nlogging: verbose: false file: path/to/log/file.log console-disabled: false cmd-std-out: false ",
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
