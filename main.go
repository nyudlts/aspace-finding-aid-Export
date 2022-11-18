package main

import (
	"flag"
	"fmt"
	"github.com/nyudlts/go-aspace"
	"log"
	"os"
	"time"
)

const appVersion = "v1.0.0b"

var (
	logfile              = "aspace-export"
	reportFile           string
	client               *aspace.ASClient
	workers              int
	config               string
	environment          string
	repository           int
	resource             int
	timeout              int
	workDir              string
	repositoryMap        map[string]int
	resourceInfo         []ResourceInfo
	validate             bool
	help                 bool
	version              bool
	reformat             bool
	format               string
	unpublishedNotes     bool
	unpublishedResources bool
	startTime            time.Time
	executionTime        time.Duration
	debug                bool
	formattedTime        string
)

type ResourceInfo struct {
	RepoID     int
	RepoSlug   string
	ResourceID int
}

func init() {
	flag.StringVar(&config, "config", "", "location of go-aspace configuration file")
	flag.StringVar(&environment, "environment", "", "environment key of instance to export from")
	flag.IntVar(&repository, "repository", 0, "ID of repository to be exported, leave blank to export all repositories")
	flag.IntVar(&resource, "resource", 0, "ID of a single resource to be exported")
	flag.IntVar(&timeout, "timeout", 20, "client timeout")
	flag.IntVar(&workers, "workers", 8, "number of concurrent workers")
	flag.BoolVar(&validate, "validate", false, "perform ead2 schema validation")
	flag.StringVar(&workDir, "export-location", "aspace-exports", "location to export finding aids")
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
	fmt.Println("  --export-location  path/to/the location to export finding aids                            	default `aspace-exports-[timestamp]`")
	fmt.Println("  --include-unpublished-notes		include unpublished notes in exports			default `false`")
	fmt.Println("  --include-unpublished-resources	include unpublished resources in exports		default `false`")
	fmt.Println("  --reformat         tab reformat ead xml files							default `false`")
	fmt.Println("  --repository       ID of the repository to be exported, `0` will export all repositories	default `0` ")
	fmt.Println("  --resource         ID of the resource to be exported, `0` will export all resources		default `0` ")
	fmt.Println("  --timeout          client timout in seconds							default 20")
	fmt.Println("  --workers          number of concurrent export workers to create				default 8")
	fmt.Println("  --validate         validate exported finding aids against ead2002 schema			default `false`")
	fmt.Println("  --debug")
	fmt.Println("  --version          print the version and version of client version")
	fmt.Println("  --help             print this help screen")
}

func main() {
	startTime = time.Now()
	formattedTime = startTime.Format("20060102-050403")
	//parse the flags
	flag.Parse()

	//check for the help message flag `--help`
	if help == true {
		printHelp()
		os.Exit(0)
	}

	//check for the version flag `--version`
	if version == true {
		fmt.Printf("aspace-export %s, using go-aspace %s\n", appVersion, aspace.LibraryVersion)
		os.Exit(0)
	}

	//starting the application
	fmt.Printf("\n-- aspace-export %s --\n\n", appVersion)

	//create a log file

	logfile = logfile + "-" + formattedTime + ".log"

	f, err := os.Create(logfile)
	if err != nil {
		printAndLog(err.Error(), FATAL)
		printHelp()
		os.Exit(1)
	}

	defer f.Close()
	log.SetOutput(f)
	printAndLog(fmt.Sprintf("logging to %s", logfile), INFO)

	//check critical flags
	err = checkFlags()
	if err != nil {
		printAndLog(err.Error(), FATAL)
		printHelp()
		os.Exit(2)
	}

	//get a go-aspace api client
	log.Println("INFO Creating go-aspace client")
	client, err = aspace.NewClient(config, environment, timeout)
	if err != nil {
		printAndLog(fmt.Sprintf("failed to create a go-aspace client %s", err.Error()), FATAL)
		os.Exit(3)
	} else {
		printAndLog(fmt.Sprintf("go-aspace client created, using go-aspace %s", aspace.LibraryVersion), INFO)
	}

	//get a map of repositories to be exported
	repositoryMap, err := getRepositoryMap()
	if err != nil {
		printAndLog(err.Error(), FATAL)
		os.Exit(4)
	}
	printAndLog(fmt.Sprintf("%d repositories returned from ArchivesSpace", len(repositoryMap)), INFO)

	//get a slice of resourceInfo
	resourceInfo, err = getResourceIDs(repositoryMap)
	if err != nil {
		printAndLog(err.Error(), FATAL)
		os.Exit(5)
	}
	printAndLog(fmt.Sprintf("%d resources returned from ArchivesSpace", len(resourceInfo)), INFO)

	//create work directory
	workDir = fmt.Sprintf("aspace-exports-%s", formattedTime)
	err = createWorkDirectory(workDir)
	if err != nil {
		printAndLog(err.Error(), FATAL)
		os.Exit(6)
	}
	printAndLog(fmt.Sprintf("working directory created at %s", workDir), INFO)

	//Create the repository export and failure directories
	err = createExportDirectories(workDir)
	if err != nil {
		printAndLog(err.Error(), FATAL)
		os.Exit(6)
	}

	//export Resources
	fmt.Printf("\nProcessing %d resources\n", len(resourceInfo))
	err = exportResources(workDir)
	if err != nil {
		printAndLog(err.Error(), FATAL)
		os.Exit(7)
	}

	//clean up directories
	err = cleanup()
	if err != nil {
		printAndLog(err.Error(), WARNING)
	}

	//exit
	printAndLog("aspace-export process complete, exiting", INFO)
	os.Exit(0)
}
