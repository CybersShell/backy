# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html),
and is generated by [Changie](https://github.com/miniscruff/changie).


## 0.2.4 - 2023-02-18
### Added
* Notifications now display errors and the output of the failed command.
* CI configs for GitHub and Woodpecker
* Added `version` subcommand
### Changed
* Console logging can be disabled by setting `console-disabled` in the `logging` object
## Fixed
* If Host was not defined for an incomplete `hosts` object, any commands would fail as they could not look up the values in the SSH config files.