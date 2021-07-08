package main

import (
	"flag"
	"fmt"
	"github.com/nyudlts/go-aspace"
	"log"
	"os"
	"path/filepath"
)

var (
	client      *aspace.ASClient
	workers		int
	config      string
	environment string
	err         error
	logfile     string
	repository  int
	timeout     int
	workDir     string
	repositoryMap map[string]int
	resourceInfo	[]ResourceInfo
)

type ResourceInfo struct {
	RepoID 		int
	RepoSlug 	string
	ResourceID	int
}

func init() {
	flag.StringVar(&config, "config", "go-aspace.yml", "location of go-aspace client")
	flag.StringVar(&environment, "environment", "", "environment key")
	flag.StringVar(&logfile, "log", "go-aspace-export.log", "location of log file")
	flag.IntVar(&repository, "repository", 0, "ID of repository to be exported, leave blank to export all repositories")
	flag.IntVar(&timeout, "timeout", 20, "client timeout")
	flag.IntVar(&workers, "workers", 8, "number of concurrent workers")
}

func main() {
	//parse the flags
	flag.Parse()
	workDir = "aspace-export"

	//create a log file
	f, err := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Printf("INFO\tRunning go-aspace-export")
	fmt.Printf("Running go-aspace finding aid exporter, logging to %s\n", logfile)

	//check critical flags
	err = checkFlags()
	if err != nil {
		fmt.Println(err.Error())
		log.Fatalln("FATAL", err)
	}

	//get a go-aspace api client
	log.Println("INFO\tRequesting API token")
	client, err = aspace.NewClient(config, environment, timeout)
	if err != nil {
		log.Fatalln("FATAL\tCould not get create an aspace client", err.Error())
	} else {
		log.Println("INFO\tgo-aspace client created, using go-aspace", aspace.LibraryVersion)
	}

	//get a map of repositories to be exported
	repositoryMap = getRepositoryMap()
	log.Printf("INFO\tfound %d repositories", len(repositoryMap))
	//get a slice of resourceInfo
	resourceInfo = []ResourceInfo{}
	getResourceIDs()

	//setup export directories
	createWorkDirectory()

	//Create the repository export and failure directories
	createExportDirectories()

	//export Resources
	log.Printf("INFO\tProcessing %d resources", len(resourceInfo))
	exportResources()

	//exit
	log.Println("INFO\tprocess complete, exiting.")
	fmt.Println("process complete, exiting.")
	os.Exit(0)
}

func checkFlags() error {
	//check if the config file exists
	if _, err := os.Stat(config); os.IsNotExist(err) {
		return fmt.Errorf("go-aspace config file does not exist at %s", config)
	}

	//check if an environment is defined
	if environment == "" {
		return fmt.Errorf("no Environment key is defined, set --environment when envoking script")
	}
	return nil
}

func createWorkDirectory() {
	if _, err = os.Stat(workDir); os.IsNotExist(err) {
		innerErr := os.Mkdir(workDir, 0777)
		if innerErr != nil {
			log.Fatalf("FATAL\tcould not create an work directory at %s", workDir)
		}
	} else {
		log.Println("INFO\twork directory exists, skipping creation", workDir)
	}
}

func createExportDirectories() {
	for slug, _ := range repositoryMap {

		repositoryDir := filepath.Join(workDir, slug)
		exportDir := filepath.Join(repositoryDir, "exports")
		failureDir := filepath.Join(repositoryDir, "failures")

		if _, err := os.Stat(repositoryDir); os.IsNotExist(err) {
			innerErr := os.Mkdir(repositoryDir, 0777)
			if innerErr != nil {
				log.Fatalf("FATAL\tcould not create a repository directory at %s", repositoryDir)
			} else {
				log.Println("INFO\tcreated repository directory", repositoryDir)
			}
		} else {
			log.Println("INFO\trepository directory exists, skipping creation of", repositoryDir)
		}

		//create the repository export directory
		if _, err := os.Stat(exportDir); os.IsNotExist(err) {
			innerErr := os.Mkdir(exportDir, 0777)
			if innerErr != nil {
				log.Fatalf("FATAL\tcould not create an exports directory at %s", exportDir)
			} else {
				log.Println("INFO\tcreated exports directory", exportDir)
			}
		} else {
			log.Println("INFO\texports directory exists, skipping creation of", exportDir)
		}

		//create the repository failure directory
		if _, err := os.Stat(failureDir); os.IsNotExist(err) {
			innerErr := os.Mkdir(failureDir, 0777)
			if innerErr != nil {
				log.Fatalf("FATAL\tcould not create a failure directory at %s", failureDir)
			} else {
				log.Println("INFO\tcreated repository directory", failureDir)
			}
		} else {
			log.Println("INFO\tfailures directory exists, skipping creation of", failureDir)
		}
	}
}

func getRepositoryMap() map[string]int {
	repositories := make(map[string]int)

	if repository != 0 {
		repositoryObject, err := client.GetRepository(repository)
		if err != nil {
			log.Fatalf("FATAL\t%s", err.Error())
		}
		repositories[repositoryObject.Slug] = repository
	} else {
		repositoryIds, err := client.GetRepositories()
		if err != nil {
			log.Fatalf("FATAL\t%s", err.Error())
		}
		for _, r := range repositoryIds {
			repositoryObject, err := client.GetRepository(r)
			if err != nil {
				log.Fatalf("FATAL\t%s", err.Error())
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
			log.Fatalf("FATAL\t%s", err.Error())
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

