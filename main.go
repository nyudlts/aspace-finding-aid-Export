package main

import (
	"flag"
	"fmt"
	"github.com/nyudlts/go-aspace"
	"io"
	"log"
	"os"
	"path/filepath"
)

var (
	logfile = "/tmp/aspace-export.log"
	client        *aspace.ASClient
	workers       int
	config        string
	environment   string
	err           error
	repository    int
	timeout       int
	workDir       string
	repositoryMap map[string]int
	resourceInfo  []ResourceInfo
	validate      bool
	help          bool
	version       bool
	appVersion    = "v0.1.0b"
)

type ResourceInfo struct {
	RepoID     int
	RepoSlug   string
	ResourceID int
}

func init() {
	flag.StringVar(&config, "config", "", "location of go-aspace configuration file")
	flag.StringVar(&environment, "environment", "dev", "environment key of instance to export from")
	flag.IntVar(&repository, "repository", 0, "ID of repository to be exported, leave blank to export all repositories")
	flag.IntVar(&timeout, "timeout", 20, "client timeout")
	flag.IntVar(&workers, "workers", 8, "number of concurrent workers")
	flag.BoolVar(&validate, "validate", false, "perform ead2 schema validation")
	flag.StringVar(&workDir, "export-location", "aspace-exports", "location to export finding aids")
	flag.BoolVar(&help, "help", false, "display the help message")
	flag.BoolVar(&version, "version", false, "display the version of the tool and go-aspace library")
}

func printHelp() {
	fmt.Println("usage: aspace-export [options]")
	fmt.Println("options:")
	fmt.Println("  --config           path/to/the go-aspace configuration file                               default `go-aspace.yml`")
	fmt.Println("  --environment      environment key in config file of the instance to export from          default `dev`")
	fmt.Println("  --repository       ID of the repository to be exported, `0` will export all repositories  default 0 -- ")
	fmt.Println("  --timeout          client timout in seconds to                                            default 20")
	fmt.Println("  --workers          number of concurrent export workers to create                          default 8")
	fmt.Println("  --validate         validate exported finding aids against ead2002 schema                  default `false`")
	fmt.Println("  --export-location  path/to/the location to export finding aids                            default `aspace-exports`")
	fmt.Println("  --help             print this help screen")
	fmt.Println("  --version          print the version and version of client version")
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
		fmt.Println("aspace-export", appVersion, "- go-aspace client", aspace.LibraryVersion)
		os.Exit(0)
	}

	//create a log file
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Printf("INFO Running go-aspace-export")
	fmt.Printf("Running go-aspace finding aid exporter, logging to %s\n", logfile)

	//check critical flags
	err = checkFlags()
	if err != nil {
		fmt.Println(err.Error())
		log.Fatalln("FATAL", err)
	}

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
	fmt.Println("Process complete, exiting.")
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

		if _, err := os.Stat(repositoryDir); os.IsNotExist(err) {
			innerErr := os.Mkdir(repositoryDir, 0777)
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
				log.Println("INFO created repository directory", failureDir)
			}
		} else {
			log.Println("INFO failures directory exists, skipping creation of", failureDir)
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
					log.Printf("INFO removing empty directory at:%s", path)
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
	newLoc := filepath.Join(workDir, "aspace-export.log")
	err = os.Rename(logfile, newLoc); if err != nil {
		fmt.Printf("Could not move log file from /tmp to %s\n", workDir)
	}

}
