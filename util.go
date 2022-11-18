package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

func getLogLevelString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "[DEBUG]"
	case INFO:
		return "[INFO]"
	case WARNING:
		return "[WARNING]"
	case ERROR:
		return "[ERROR]"
	case FATAL:
		return "[FATAL]"
	default:
		panic(fmt.Errorf("log level %v is not supported", level))
	}
}

func printAndLog(msg string, logLevel LogLevel) {
	if logLevel == DEBUG && debug == false {

	} else {
		level := getLogLevelString(logLevel)
		fmt.Printf("%s %s\n", level, msg)
		log.Printf("%s %s", level, msg)
	}
}

func checkFlags() error {
	//check if the config file exists
	if config == "" {
		return fmt.Errorf("location of go-aspace config file is mandatory, set the --config option when running aspace-export")
	}

	if _, err := os.Stat(config); os.IsNotExist(err) {
		return fmt.Errorf("go-aspace config file does not exist at %s", config)
	}

	if environment == "" {
		return fmt.Errorf("environment to run export against is mandatory, set the --env option when running aspace=export")
	}

	if format != "marc" && format != "ead" {
		return fmt.Errorf("format must be either `ead` or `marc`, set the --format option when running aspace-export")
	}

	if resource != 0 && repository == 0 {
		return fmt.Errorf("a single resource can not be exported if the repository is not specified, set the --repository option when running aspace-export")
	}

	return nil
}

func createWorkDirectory() error {
	if _, err = os.Stat(workDir); os.IsNotExist(err) {
		innerErr := os.Mkdir(workDir, 0777)
		if innerErr != nil {
			return innerErr
		}
	}

	if err != nil {
		return err
	} else {
		return nil
	}
}

func createExportDirectories() {
	for slug := range repositoryMap {

		repositoryDir := filepath.Join(workDir, slug)
		exportDir := filepath.Join(repositoryDir, "exports")
		failureDir := filepath.Join(repositoryDir, "invalid")
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

func getRepositoryMap() (map[string]int, error) {
	repositories := make(map[string]int)

	if repository != 0 {
		//export a single repository
		repositoryObject, err := client.GetRepository(repository)
		if err != nil {
			return repositories, fmt.Errorf("repository id %d does not exist in the %s environment", repository, environment)
		}
		repositories[repositoryObject.Slug] = repository
	} else {
		//export all repositories
		repositoryIds, err := client.GetRepositories()
		if err != nil {
			return repositories, err
		}
		for _, r := range repositoryIds {
			repositoryObject, err := client.GetRepository(r)
			if err != nil {
				return repositories, err
			}
			repositories[repositoryObject.Slug] = r
		}
	}
	return repositories, nil
}

func getResourceIDs(repMap map[string]int) ([]ResourceInfo, error) {

	resources := []ResourceInfo{}

	for repositorySlug, repositoryID := range repMap {
		if resource != 0 {
			resources = append(resources, ResourceInfo{
				RepoID:     repositoryID,
				RepoSlug:   repositorySlug,
				ResourceID: resource,
			})
			continue
		}

		resourceIDs, err := client.GetResourceIDs(repositoryID)
		if err != nil {
			return resources, err
		}

		for _, resourceID := range resourceIDs {
			resources = append(resources, ResourceInfo{
				RepoID:     repositoryID,
				RepoSlug:   repositorySlug,
				ResourceID: resourceID,
			})
		}
	}

	return resources, nil
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

	//move the reportFile to the workdir
	newLoc = filepath.Join(workDir, reportFile)
	err = os.Rename(reportFile, newLoc)
	if err != nil {
		fmt.Println(err.Error())
	}

}
