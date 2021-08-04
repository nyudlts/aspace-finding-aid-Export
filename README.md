aspace-export
=============
Command-line utility for bulk export, validation and reformatting of EAD finding aids from Archivespace.

Install
-------
1. download the latest binary for Mac or linux
2. decompress the zip file
3. Enter your ArchivesSpace credentials into the go-aspace.yml file

Run
---
$ aspace-export --config /path/to/go-aspace.yml [options] 2> /dev/null 
<br><br><b>note:</b> the underlying xml lib will output a lot of info about xml errors to stderr, `2> /dev/null` ignores the output but you can redirect to a file by replacing /dev/null 

The program will create a directory hierarchy at the location set in the --export-location option. There will be a subdirectory created for each repository that was exported, with the name of the repositories slug.
within each repository directory there will be an `exports` directory containing all exported finding aids. 
If the `validate` option was set when the program was run any finding aids that fail validation will be written to a subdirectory named `failures`.
A log file will be created named `aspace-export.log` which will be moved to the root of output directory at the end of the process, the initial location of this log file can be set with the `--logfile` option.


**example output structure**<br>
aspace-exports<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;aspace-exports.log<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;/tamwag<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;/exports<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;tam_001.xml<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;tam_002.xml<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;/failures<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;tam_003.xml<br>

**note**</br>
* The program currently uses a resource's `eadid` to create a filename, if the resource's eadid is blank it will be skipped and marked in the logfile.<br>
* The program currently only exports resources that have a `Publish` value set to `true`<br>

Command-Line Arguments
----------------------
--config, path/to/go-aspace.yml configuration file, required<br>
--logfile, path/to/the logfile to be created, default `aspace-export.log`<br>
--environment, environment key in config file of the instance to export from, default: `dev`<br>
--repository, ID of the repsoitory to be exported, `0` will export all repositories, default: 0<br>
--timeout, client timout in seconds to, default: 20<br>
--workers, number of concurrent export workers to create, default: 8<br>
--validate, validate exported finding aids against ead2002 schema, default: `false`<br>
--reformat, tab-reformat ead files, default: `false`<br/>
--export-location, path/to/the location to export finding aids, default: `./aspace-exports`<br>
--help, print this help screen<br>
--version, print the application and go-aspace client version
