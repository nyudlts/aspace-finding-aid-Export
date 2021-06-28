package main

import (
	"flag"
	"fmt"
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
	setupDirectories("exports", "failures")

	//get a map of repositories to be exported
	repositories := getRepositoryMap(repository)

	//export the repositories
	for slug, id := range repositories {
		fmt.Printf("Exporting repository: %s\n", slug)
		err := exportRepository(slug, id)
		if err != nil {
			log.Println("ERROR", err.Error())
		}
	}
}

func checkFlags() error {
	//check if the config file exists
	if _, err := os.Stat(config); os.IsNotExist(err) {
		return fmt.Errorf("Go-aspace config file does not exist at %s", config)
	}

	//check if an environment is defined
	if environment == "" {
		return fmt.Errorf("No Environment key is defined, set --environment when envoking script.")
	}
	return nil
}

func setupDirectories(exports string, failures string) {
	if _, err = os.Stat(exports); os.IsNotExist(err) {
		inner_err := os.Mkdir(exports, 0666)
		if inner_err != nil {
			log.Fatalln("could not create an exports directory at %s", exports)
		}
	} else {
		log.Println("INFO", "exports directory exists, skipping creation")
	}

	if _, err = os.Stat(failures); os.IsNotExist(err) {
		inner_err := os.Mkdir(failures, 066)
		if inner_err != nil {
			log.Fatalln("ERROR", "could not create a failures directory at %s", failures)
		}
	} else {
		log.Println("INFO", "failures directory exists, skipping creation")
	}
}

func getRepositoryMap(repId int) map[string]int {
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
	return nil
}
