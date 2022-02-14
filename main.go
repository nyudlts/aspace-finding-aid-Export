package main

import (
	"flag"
	"fmt"
	"github.com/nyudlts/go-aspace"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	logfile              string
	client               *aspace.ASClient
	workers              int
	config               string
	environment          string
	err                  error
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
	appVersion           = "v0.5.0b"
)

type ResourceInfo struct {
	RepoID     int
	RepoSlug   string
	ResourceID int
}

func init() {
	flag.StringVar(&config, "config", "", "location of go-aspace configuration file")
	flag.StringVar(&logfile, "logfile", "aspace-export", "location of the log file to be written")
	flag.StringVar(&environment, "environment", "dev", "environment key of instance to export from")
	flag.IntVar(&repository, "repository", 0, "ID of repository to be exported, leave blank to export all repositories")
	flag.IntVar(&resource, "resource", 0, "ID of a single resource to be exported")
	flag.IntVar(&timeout, "timeout", 20, "client timeout")
	flag.IntVar(&workers, "workers", 8, "number of concurrent workers")
	flag.BoolVar(&validate, "validate", false, "perform ead2 schema validation")
	flag.StringVar(&workDir, "export-location", "aspace-exports", "location to export finding aids")
	flag.BoolVar(&help, "help", false, "display the help message")
	flag.BoolVar(&version, "version", false, "display the version of the tool and go-aspace library")
	flag.BoolVar(&reformat, "reformat", false, "tab reformat the output file")
	flag.StringVar(&format, "format", "ead", "format of export: ead or marc")
	flag.BoolVar(&unpublishedNotes, "include-unpublished-notes", false, "include unpublished notes")
	flag.BoolVar(&unpublishedResources, "include-unpublished-resources", false, "include unpublished resources")
}

func printHelp() {
	fmt.Println("usage: aspace-export [options]")
	fmt.Println("options:")
	fmt.Println("  --config           path/to/the go-aspace configuration file                               default `go-aspace.yml`")
	fmt.Println("  --environment      environment key in config file of the instance to export from          default `dev`")
	fmt.Println("  --export-location  path/to/the location to export finding aids                            default `aspace-exports`")
	fmt.Println("  --format           the export format either `ead` or `marc`                               default `ead`")
	fmt.Println("  --include-unpublished-notes		inlude unpublished notes in exports					 default `false`")
	fmt.Println("  --include-unpublished-resources	inlude unpublished resources in exports				 default `false`")
	fmt.Println("  --logfile          path/to/the logfile                                                    default `aspace-export-yyyymmdd.log`")
	fmt.Println("  --reformat         tab reformat ead xml files                                             default `false`")
	fmt.Println("  --repository       ID of the repository to be exported, `0` will export all repositories  default 0 -- ")
	fmt.Println("  --resource         ID of the resource to be exported, `0` will export all resources  	 default 0 -- ")
	fmt.Println("  --timeout          client timout in seconds                                               default 20")
	fmt.Println("  --workers          number of concurrent export workers to create                          default 8")
	fmt.Println("  --validate         validate exported finding aids against ead2002 schema                  default `false`")
	fmt.Println("  --version          print the version and version of client version")
	fmt.Println("  --help             print this help screen")
}

func main() {
	//parse the flags
	flag.Parse()

	//check for the help message flag
	if help == true {
		printHelp()
		os.Exit(0)
	}

	//check for the version flag
	if version == true {
		fmt.Printf("aspace-export %s, using go-aspace %s\n", appVersion, aspace.LibraryVersion)
		os.Exit(0)
	}

	//create a log file
	t := time.Now()
	tf := t.Format("20060102")
	logfile = logfile + "-" + tf + ".log"
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Printf("INFO Running go-aspace-export")

	//check critical flags
	err = checkFlags()
	if err != nil {
		fmt.Println("FATAL error:", err.Error())
		log.Fatalln("FATAL", err)
	}

	fmt.Printf("Running go-aspace %s exporter, logging to %s\n", format, logfile)

	//get a go-aspace api client
	log.Println("INFO Creating go-aspace client")
	client, err = aspace.NewClient(config, environment, timeout)
	if err != nil {
		log.Fatalln("FATAL Could not get create an aspace client", err.Error())
	} else {
		log.Println("INFO go-aspace client created, using go-aspace", aspace.LibraryVersion)
	}

	//get a map of repositories to be exported
	repositoryMap = getRepositoryMap()
	log.Printf("INFO found %d repositories", len(repositoryMap))
	//get a slice of resourceInfo
	resourceInfo = []ResourceInfo{}
	getResourceIDs()

	//setup export directories
	createWorkDirectory()

	//Create the repository export and failure directories
	createExportDirectories()

	//export Resources
	fmt.Printf("Processing %d resources\n", len(resourceInfo))
	exportResources()

	//clean up directories
	cleanup()

	//exit
	log.Println("INFO process complete, exiting.")
	fmt.Println("\naspace-export complete, exiting.")
	os.Exit(0)
}

func checkFlags() error {
	//check if the config file exists
	if config == "" {
		return fmt.Errorf("location of go-aspace config file is mandatory, set the --config option")
	}

	if _, err := os.Stat(config); os.IsNotExist(err) {
		return fmt.Errorf("go-aspace config file does not exist at %s", config)
	}

	if format != "marc" && format != "ead" {
		return fmt.Errorf("--format must be either ead or marc")
	}

	if resource != 0 && repository == 0 {
		return fmt.Errorf("A single resource can not be exported if the repository is not specified, include the --repository flag.")
	}

	return nil
}

func createWorkDirectory() {
	if _, err = os.Stat(workDir); os.IsNotExist(err) {
		innerErr := os.Mkdir(workDir, 0777)
		if innerErr != nil {
			log.Fatalf("FATAL could not create an work directory at %s", workDir)
		}
	} else {
		log.Println("INFO work directory exists, skipping creation", workDir)
	}
}

func createExportDirectories() {
	for slug := range repositoryMap {

		repositoryDir := filepath.Join(workDir, slug)
		exportDir := filepath.Join(repositoryDir, "exports")
		failureDir := filepath.Join(repositoryDir, "failures")
		unpublishedDir := filepath.Join(repositoryDir, "unpublished")

		if _, err := os.Stat(repositoryDir); os.IsNotExist(err) {
			innerErr := os.Mkdir(repositoryDir, 0755)
			if innerErr != nil {
				log.Fatalf("FATAL could not create a repository directory at %s", repositoryDir)
			} else {
				log.Println("INFO created repository directory", repositoryDir)
			}
		} else {
			log.Println("INFO repository directory exists, skipping creation of", repositoryDir)
		}

		//create the repository export directory
		if _, err := os.Stat(exportDir); os.IsNotExist(err) {
			innerErr := os.Mkdir(exportDir, 0777)
			if innerErr != nil {
				log.Fatalf("FATAL could not create an exports directory at %s", exportDir)
			} else {
				log.Println("INFO created exports directory", exportDir)
			}
		} else {
			log.Println("INFO exports directory exists, skipping creation of", exportDir)
		}

		//create the repository failure directory
		if _, err := os.Stat(failureDir); os.IsNotExist(err) {
			innerErr := os.Mkdir(failureDir, 0777)
			if innerErr != nil {
				log.Fatalf("FATAL could not create a failure directory at %s", failureDir)
			} else {
				log.Println("INFO created failures directory", failureDir)
			}
		} else {
			log.Println("INFO failures directory exists, skipping creation of", failureDir)
		}

		if unpublishedResources == true {
			if _, err := os.Stat(unpublishedDir); os.IsNotExist(err) {
				innerErr := os.Mkdir(unpublishedDir, 0777)
				if innerErr != nil {
					log.Fatalf("FATAL could not create a unpublished directory at %s", unpublishedDir)
				} else {
					log.Println("INFO created unpublished directory", unpublishedDir)
				}
			} else {
				log.Println("INFO unpublished directory exists, skipping creation of", unpublishedDir)
			}
		}
	}
}

func getRepositoryMap() map[string]int {
	repositories := make(map[string]int)

	if repository != 0 {
		//export a single repository
		repositoryObject, err := client.GetRepository(repository)
		if err != nil {
			fmt.Printf("Repository id %d does not exist in %s instance\n", repository, environment)
			log.Fatalf("FATAL %s", err.Error())
		}
		repositories[repositoryObject.Slug] = repository
	} else {
		//export all repositories
		repositoryIds, err := client.GetRepositories()
		if err != nil {
			log.Fatalf("FATAL %s", err.Error())
		}
		for _, r := range repositoryIds {
			repositoryObject, err := client.GetRepository(r)
			if err != nil {
				log.Fatalf("FATAL %s", err.Error())
			}
			repositories[repositoryObject.Slug] = r
		}
	}
	return repositories
}

func getResourceIDs() {

	for repositorySlug, repositoryID := range repositoryMap {
		resourceIDs, err := client.GetResourceIDs(repositoryID)
		if err != nil {
			log.Fatalf("FATAL %s", err.Error())
		}
		if resource != 0 {
			resourceInfo = append(resourceInfo, ResourceInfo{
				RepoID:     repository,
				RepoSlug:   repositorySlug,
				ResourceID: resource,
			})
			continue
		}
		for _, resourceID := range resourceIDs {
			resourceInfo = append(resourceInfo, ResourceInfo{
				RepoID:     repositoryID,
				RepoSlug:   repositorySlug,
				ResourceID: resourceID,
			})
		}
	}
}

func cleanup() {
	//remove any empty directories
	err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				log.Println("ERROR ", err.Error())
			} else {
				defer f.Close()
				_, err = f.Readdirnames(1)
				if err == io.EOF {
					log.Printf("INFO removing empty directory at: %s", path)
					os.Remove(path)
				}
			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	//move the logfile to the workdir
	newLoc := filepath.Join(workDir, logfile)
	err = os.Rename(logfile, newLoc)
	if err != nil {
		fmt.Println(err.Error())
	}

}
