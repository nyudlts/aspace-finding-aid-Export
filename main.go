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
	config      string
	environment string
	err         error
	logfile     string
	repository  int
	timeout     int
	workDir     string
)

func init() {
	flag.StringVar(&config, "config", "go-aspace.yml", "location of go-aspace client")
	flag.StringVar(&environment, "environment", "", "environemnt key")
	flag.StringVar(&logfile, "log", "go-aspace-export.log", "location of log file")
	flag.IntVar(&repository, "repository", 0, "ID of repository to be exported, leave blank to export all repositories")
	flag.IntVar(&timeout, "timeout", 20, "client timeout")
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
	log.Println("INFO", "Running go-aspace-export")
	fmt.Printf("Running go-aspace finding aid exporter, logging to %s\n", logfile)

	//check critical flags
	err = checkFlags()
	if err != nil {
		fmt.Println(err.Error())
		log.Fatalln("FATAL", err)
	}

	//get a go-aspace api client
	log.Println("INFO", "Requesting API token")
	client, err = aspace.NewClient(config, environment, timeout)
	if err != nil {
		log.Fatalln("FATAL", "Could not get create an aspace client", err.Error())
	} else {
		log.Println("INFO", "go-aspace client created, using go-aspace", aspace.LibraryVersion)
	}

	//setup export directories
	setupDirectories()

	//get a map of repositories to be exported
	repositories := getRepositoryMap()

	//export the repositories
	for slug, id := range repositories {
		fmt.Printf("Exporting repository: %s\n", slug)
		err := exportRepository(slug, id)
		if err != nil {
			log.Println("ERROR", err.Error())
		}
	}

	//exit
	fmt.Println("Process Complete, exiting.")
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

func setupDirectories() {

	if _, err = os.Stat(workDir); os.IsNotExist(err) {
		innerErr := os.Mkdir(workDir, 0777)
		if innerErr != nil {
			log.Fatalf("could not create an work directory at %s", workDir)
		}
	} else {
		log.Println("INFO", "work directory exists, skipping creation", workDir)
	}
}

func getRepositoryMap() map[string]int {
	repositories := make(map[string]int)

	if repository != 0 {
		repositoryObject, err := client.GetRepository(repository)
		if err != nil {
			log.Fatalln("FATAL", err.Error())
		}
		repositories[repositoryObject.Slug] = repository
	} else {
		repositoryIds, err := client.GetRepositories()
		if err != nil {
			log.Fatalln("FATAL", err.Error())
		}
		for _, r := range repositoryIds {
			repositoryObject, err := client.GetRepository(r)
			if err != nil {
				log.Fatalln("FATAL", err.Error())
			}
			repositories[repositoryObject.Slug] = r
		}
	}
	return repositories
}

func exportRepository(slug string, id int) error {
	repositoryDir := filepath.Join(workDir, slug)
	exportDir := filepath.Join(repositoryDir, "exports")
	failureDir := filepath.Join(repositoryDir, "failures")

	//create the repository directory
	if _, err := os.Stat(repositoryDir); os.IsNotExist(err) {
		innerErr := os.Mkdir(repositoryDir, 0777)
		if innerErr != nil {
			log.Fatalf("FATAL could not create a repository directory at %s", repositoryDir)
		} else {
			log.Println("INFO", "created repository directory", repositoryDir)
		}
	} else {
		log.Println("INFO", "repository directory exists, skipping creation of", repositoryDir)
	}

	//create the repository export directory
	if _, err := os.Stat(exportDir); os.IsNotExist(err) {
		innerErr := os.Mkdir(exportDir, 0777)
		if innerErr != nil {
			log.Fatalf("FATAL could not create an exports directory at %s", exportDir)
		} else {
			log.Println("INFO", "created exports directory", exportDir)
		}
	} else {
		log.Println("INFO", "exports directory exists, skipping creation of", exportDir)
	}

	//create the repository failure directory
	if _, err := os.Stat(failureDir); os.IsNotExist(err) {
		innerErr := os.Mkdir(failureDir, 0777)
		if innerErr != nil {
			log.Fatalf("FATAL could not create a failure directory at %s", failureDir)
		} else {
			log.Println("INFO", "created repository directory", failureDir)
		}
	} else {
		log.Println("INFO", "failures directory exists, skipping creation of", failureDir)
	}

	return nil

}
