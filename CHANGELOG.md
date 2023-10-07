# Versioning policy

We are following [Semantic versioning](https://semver.org/) in this library

We will mark these with Git Tags

# Changes in releases

## release 1.1.0

Bugfixes:  
-

New features:  
 * New log level NoneLevel (configured with "none" or "off") is introduced to make a certain logger (and its hierarchy) completely silent
 * Logger now can tell if a specific level is enabled or not. Methods like .IsInfoEnabled() are available
 * In config Yaml/Json from now the levels are case insensitive - writing "level: INFO" or "level: info" or "level: Info" does not matter anymore
 * In config Yaml/Json from now "level: warn" is also accepted and this is the same as "level: warning"

Breaking changes:  
-


## release 1.0.3

Bugfixes:
 * fix panic if no logger is initialised
 * fix default logger to log to console

New features:  
-

Breaking changes:  
-

## release 1.0.2

Bugfixes:
 * .WithLabels() and .WithLabel() had bug - it did not add the provided label(s) - this was fixed

New features:  
-

Breaking changes:  
-

## release 1.0.1

Bugfixes:
 * Thread safety problem fix - protecting GetLogger() with mutex

New features:  
- 

Breaking changes:  
-

## release 1.0.0

Bugfixes:  
-

New features:
 * as this is the initial commit - everything ;-)

Breaking changes:  
-
