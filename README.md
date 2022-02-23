aspace-export, v0.5.1b
=============
Command-line utility for bulk export, validation and reformatting of EAD finding aids and MARC recordsfrom Archivespace.

Install From Binary
-------------------
1. Download the latest binary for Mac or linux https://github.com/nyudlts/aspace-export/releases/tag/v0.5.1b
3. Enter your ArchivesSpace credentials into the go-aspace.yml file included in the zip.

Install With Go
---------------
$ go install github.com/nyudlts/aspace-export

Build From Source
-----------------
Pre-requisite: libxml2<br>
$ make build 


Run
---
$ aspace-export --config /path/to/go-aspace.yml [options] 
<br><br>**notes:** 

* The underlying xml lib will output voluminous, and not always helpful, info about xml errors to stderr, `2> /dev/null` ignores the output but you can redirect to a file by replacing /dev/null
* The program will create a directory hierarchy at the location set in the --export-location option. There will be a subdirectory created for each repository that was exported, with the name of the repositories slug.
within each repository directory there will be an `exports` directory containing all exported finding aids. 
* If the `validate` option is set when the running the application any finding aids that fail validation will be written to a subdirectory named `invalid`.
* A log file will be created named `aspace-export-[yyyymmdd].log` which will be moved to the root of output directory at the end of the process.
* A Report with statistics will be created named `aspace-export-[yyyymmdd]-report.txt` which will be moved to the root of output directory at the end of the process.
* Validation of MARC21 records is currently failing, do not use the --marc flag in conjunction with the --validate flage


**example output structure**<br>
aspace-exports<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;aspace-exports.log<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;/tamwag<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;/exports<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;tam_001.xml<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;tam_002.xml<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;/invalid<br>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;tam_003.xml<br>

Command-Line Arguments
----------------------
--config, path/to/go-aspace.yml configuration file, required<br>
--environment, environment key in config file of the instance to export from, default: `dev`<br>
--export-location, path/to/the location to export resources, default: `./aspace-exports`<br>
--format, format of export: ead or marc, default: `ead`<br>
--include-unpublished-resources, include unpublished resources in exports, default: false<br>
--include-unpublished-notes, include unpublished notes in exports, default: false<br>
--reformat, tab-reformat ead files (marcxml are tab-formatted by ArchivesSpace), default: `false`<br>
--repository, ID of the repository to be exported, `0` will export all repositories, default: 0<br>
--resource, ID of the resource to be exported, `0` will export all resources, default: 0<br>
--timeout, client timout in seconds to, default: 20<br>
--validate, validate exported finding aids against schema, default: `false`<br>
--version, print the application and go-aspace client version
--workers, number of concurrent export workers to create, default: 8<br>
--help, print this help screen<br>

Package Distrobution
-----------------
Pre-requisite: libxml2<br>
$ make package VERSION="release version" OS="osx,linux"

