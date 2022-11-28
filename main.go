package main

import (
	"flag"
	"fmt"
	export "github.com/nyudlts/aspace-export/aspace_xport"
	"github.com/nyudlts/go-aspace"
	"os"
	"path/filepath"
	"time"
)

const appVersion = "v1.0.0b"

var (
	config               string
	debug                bool
	environment          string
	formattedTime        string
	format               string
	help                 bool
	reformat             bool
	repository           int
	resource             int
	resourceInfo         []export.ResourceInfo
	startTime            time.Time
	timeout              int
	unpublishedNotes     bool
	unpublishedResources bool
	validate             bool
	version              bool
	workDir              string
	workers              int
)

func init() {
	flag.StringVar(&config, "config", "", "location of go-aspace configuration file")
	flag.StringVar(&environment, "environment", "", "environment key of instance to export from")
	flag.IntVar(&repository, "repository", 0, "ID of repository to be exported, leave blank to export all repositories")
	flag.IntVar(&resource, "resource", 0, "ID of a single resource to be exported")
	flag.IntVar(&timeout, "timeout", 20, "client timeout")
	flag.IntVar(&workers, "workers", 8, "number of concurrent workers")
	flag.BoolVar(&validate, "validate", false, "perform ead2 schema validation")
	flag.StringVar(&workDir, "export-location", ".", "location to export finding aids")
	flag.BoolVar(&help, "help", false, "display the help message")
	flag.BoolVar(&version, "version", false, "display the version of the tool and go-aspace library")
	flag.BoolVar(&reformat, "reformat", false, "tab reformat the output file")
	flag.StringVar(&format, "format", "", "format of export: ead or marc")
	flag.BoolVar(&unpublishedNotes, "include-unpublished-notes", false, "include unpublished notes")
	flag.BoolVar(&unpublishedResources, "include-unpublished-resources", false, "include unpublished resources")
	flag.BoolVar(&debug, "debug", false, "")
}

func printHelp() {
	fmt.Println("usage: aspace-export [options]")
	fmt.Println("options:")
	fmt.Println("  --config           path/to/the go-aspace configuration file					mandatory")
	fmt.Println("  --environment      environment key in config file of the instance to run export against   	mandatory")
	fmt.Println("  --format           the export format either `ead` or `marc					mandatory")
	fmt.Println("  --export-location  path/to/the location to export finding aids                            	default `.`")
	fmt.Println("  --include-unpublished-notes		include unpublished notes in exports			default `false`")
	fmt.Println("  --include-unpublished-resources	include unpublished resources in exports		default `false`")
	fmt.Println("  --reformat         tab reformat ead xml files							default `false`")
	fmt.Println("  --repository       ID of the repository to be exported, `0` will export all repositories	default `0` ")
	fmt.Println("  --resource         ID of the resource to be exported, `0` will export all resources		default `0` ")
	fmt.Println("  --timeout          client timout in seconds							default `20`")
	fmt.Println("  --workers          number of concurrent export workers to create				default `8`")
	fmt.Println("  --validate         validate exported finding aids against ead2002 schema			default `false`")
	fmt.Println("  --debug	     print debug messages							default `false`")
	fmt.Println("  --version          print the version and version of client version\n")
}

func main() {

	//parse the flags
	flag.Parse()

	//check for the help message flag `--help`
	if help == true {
		printHelp()
		os.Exit(0)
	}

	//check for the version flag `--version`
	if version == true {
		fmt.Printf("  aspace-export %s <https://github.com/nyudlts/aspace-export>\n", appVersion)
		fmt.Printf("  go-aspace %s <https://github.com/nyudlts/go-aspace>\n", aspace.LibraryVersion)
		os.Exit(0)
	}

	//create timestamp for files
	startTime = time.Now()
	formattedTime = startTime.Format("20060102-050403")

	//starting the application
	export.PrintOnly(fmt.Sprintf("aspace-export %s", appVersion), export.INFO)

	//create logger
	err := export.CreateLogger(debug)
	if err != nil {
		export.PrintAndLog(err.Error(), export.ERROR)
		printHelp()
		os.Exit(1)
	}
	export.LogOnly(fmt.Sprintf("aspace-export %s", appVersion), export.INFO)

	//check critical flags
	err = export.CheckFlags(config, environment, format, resource, repository)
	if err != nil {
		export.PrintAndLog(err.Error(), export.FATAL)
		err = export.CloseLogger()
		if err != nil {
			export.PrintAndLog(err.Error(), export.ERROR)
		}
		printHelp()
		os.Exit(2)
	}
	export.PrintAndLog("all mandatory options set", export.INFO)

	//check that export location exists
	err = export.CheckPath(workDir)
	if err != nil {
		export.PrintAndLog(err.Error(), export.FATAL)
		err = export.CloseLogger()
		if err != nil {
			export.PrintAndLog(err.Error(), export.ERROR)
		}
		os.Exit(3)
	}
	export.PrintAndLog(fmt.Sprintf("%s exists and is a directory", workDir), export.INFO)

	//get a go-aspace api client
	export.PrintOnly("Creating go-aspace client", export.INFO)
	err = export.CreateAspaceClient(config, environment, timeout)
	if err != nil {
		export.PrintAndLog(fmt.Sprintf("failed to create a go-aspace client %s", err.Error()), export.FATAL)
		err = export.CloseLogger()
		if err != nil {
			export.PrintAndLog(err.Error(), export.ERROR)
		}
		os.Exit(4)
	} else {
		export.PrintAndLog(fmt.Sprintf("go-aspace client created, using go-aspace %s", aspace.LibraryVersion), export.INFO)
	}

	//get a map of repositories to be exported
	repositoryMap, err := export.GetRepositoryMap(repository, environment)
	if err != nil {
		export.PrintAndLog(err.Error(), export.FATAL)
		err = export.CloseLogger()
		if err != nil {
			export.PrintAndLog(err.Error(), export.ERROR)
		}
		os.Exit(5)
	}
	export.PrintAndLog(fmt.Sprintf("%d repositories returned from ArchivesSpace", len(repositoryMap)), export.INFO)

	//get a slice of resourceInfo
	resourceInfo, err = export.GetResourceIDs(repositoryMap, resource)
	if err != nil {
		export.PrintAndLog(err.Error(), export.FATAL)
		err = export.CloseLogger()
		if err != nil {
			export.PrintAndLog(err.Error(), export.ERROR)
		}
		os.Exit(6)
	}
	export.PrintAndLog(fmt.Sprintf("%d resources returned from ArchivesSpace", len(resourceInfo)), export.INFO)

	//create work directory
	workDir = filepath.Join(workDir, fmt.Sprintf("aspace-exports-%s", formattedTime))
	err = export.CreateWorkDirectory(workDir)
	if err != nil {
		export.PrintAndLog(err.Error(), export.FATAL)
		err = export.CloseLogger()
		if err != nil {
			export.PrintAndLog(err.Error(), export.ERROR)
		}
		os.Exit(7)
	}
	export.PrintAndLog(fmt.Sprintf("working directory created at %s", workDir), export.INFO)

	//Create the repository export and failure directories
	err = export.CreateExportDirectories(workDir, repositoryMap, unpublishedResources)
	if err != nil {
		export.PrintAndLog(err.Error(), export.FATAL)
		err = export.CloseLogger()
		if err != nil {
			export.PrintAndLog(err.Error(), export.ERROR)
		}
		os.Exit(8)
	}

	//Validate the export format
	xportFormat, err := export.GetExportFormat(format)
	if err != nil {
		export.PrintAndLog(err.Error(), export.FATAL)
		err = export.CloseLogger()
		if err != nil {
			export.PrintAndLog(err.Error(), export.ERROR)
		}
		os.Exit(9)
	}

	//create ExportOptions struct
	xportOptions := export.ExportOptions{
		WorkDir:              workDir,
		Format:               xportFormat,
		UnpublishedNotes:     unpublishedNotes,
		UnpublishedResources: unpublishedResources,
		Validate:             validate,
		Workers:              workers,
		Reformat:             reformat,
	}

	//export resources
	export.PrintAndLog(fmt.Sprintf("processing %d resources", len(resourceInfo)), export.INFO)
	err = export.ExportResources(xportOptions, startTime, formattedTime, &resourceInfo)
	if err != nil {
		export.PrintAndLog(err.Error(), export.FATAL)
		err = export.CloseLogger()
		if err != nil {
			export.PrintAndLog(err.Error(), export.ERROR)
		}
		os.Exit(10)
	}

	//clean up directories
	err = export.Cleanup(workDir)
	if err != nil {
		export.PrintAndLog(err.Error(), export.WARNING)
	}

	//exit
	export.PrintAndLog("aspace-export process complete, exiting\n", export.INFO)
	err = export.CloseLogger()
	if err != nil {
		export.PrintAndLog(err.Error(), export.ERROR)
	}

	os.Exit(0)
}
