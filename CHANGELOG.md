# Versioning policy

We are following [Semantic versioning](https://semver.org/) in this library

We will mark these with Git Tags

# Changes in releases

## release 2.0.0

Breaking changes:

- Restructuring packages so no package aliasing needed anymore

New features:

- From now on we use `DisableCaller` and `DisableStacktrace` - we never used it anyways
- Adding support for rotated files so disk usage / how long we keep logs can be controlled (provided by https://github.com/natefinch/lumberjack)

Other changes:

- Switching to go 1.23.4
- Added code formatter
- Upgraded dependencies to latest available

Known issues / limitations:

- Problem: On Windows you will exprience "The process cannot access the file because it is being used by another process." if you use rolling file setup.
- Reason: Because we are using lumberjack lib for rotation, we inherit this issue: https://github.com/natefinch/lumberjack/issues/185
- Solution: On Windows there is no good / easy solution so unfortunately this can not be used on Windows

## release 1.1.0

New features:

- New log level NoneLevel (configured with "none" or "off") is introduced to make a certain logger (and its hierarchy) completely silent
- Logger now can tell if a specific level is enabled or not. Methods like .IsInfoEnabled() are available
- In config Yaml/Json from now the levels are case insensitive - writing "level: INFO" or "level: info" or "level: Info" does not matter anymore
- In config Yaml/Json from now "level: warn" is also accepted and this is the same as "level: warning"

## Breaking changes:

## release 1.0.3

Bugfixes:

- fix panic if no logger is initialised
- fix default logger to log to console

## release 1.0.2

Bugfixes:

- .WithLabels() and .WithLabel() had bug - it did not add the provided label(s) - this was fixed

## release 1.0.1

Bugfixes:

- Thread safety problem fix - protecting GetLogger() with mutex

## release 1.0.0

## Bugfixes:

New features:

- as this is the initial commit - everything ;-)
