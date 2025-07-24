# Running commands on hosts

I want all commands in a list to be able to be run on all hosts. The underlying solution will be using a function to run the list on a host, and therefore change the host on the commands. This can be done in several ways:

1. The commands can have a `Hosts` field that will be a []string. This array can be populated several ways:
    - From the config file
    - using CLI options and commands
   The commands can be run in succession on all hosts using functions

2. The existing `Host` field can be modified in a function. The commands need to be added to a `[]*Command` slice so that all hosts can be run.
