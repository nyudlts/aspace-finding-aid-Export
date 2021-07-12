aspace-export
=============
Command-line utility to bulk export EAD finding aids from Archivespace.

Install
-------

Run
---
$ aspace-export [options]

Command-Line Arguments
----------------------
--config, path/to/the go-aspace configuration file, default: `go-aspace.yml`<br>
--environment, environment key in config file of the instance to export from, default: `dev`<br>
--log, path/to/the log file to be created by the application, default: `go-aspace.yml`<br>
--repository, ID of the repsoitory to be exported, `0` will export all repositories, default: 0<br>
--timeout, client timout in seconds to, default: 20<br>
--workers, number of concurrent export workers to create, default: 8<br>
--validate, validate exported finding aids against ead2002 schema, default: `false`<br>
--export-location, path/to/the location to export finding aids, default: `aspace-exports`<br>
--help, print this help screen<br>, --version, print the version and version of client version<br
