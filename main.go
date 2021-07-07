package main

import (
	"flag"
	"fmt"
	"github.com/dmnyu/aspace-fa-export/export"
	"github.com/nyudlts/go-aspace"
	"log"
	"os"
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

	//setup export directories
	setupDirectories()

	//get a map of repositories to be exported
	repositories := getRepositoryMap()

	//export the repositories
	for slug, id := range repositories {
		fmt.Printf("Exporting repository: %s\n", slug)
		err := export.ExportRepository(slug, id, workDir, client)
		if err != nil {
			log.Printf("ERROR\t%s", err.Error())
		}
	}

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

func setupDirectories() {

	if _, err = os.Stat(workDir); os.IsNotExist(err) {
		innerErr := os.Mkdir(workDir, 0777)
		if innerErr != nil {
			log.Fatalf("FATAL\tcould not create an work directory at %s", workDir)
		}
	} else {
		log.Println("INFO\twork directory exists, skipping creation", workDir)
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
