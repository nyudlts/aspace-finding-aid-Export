package aspace_xport

import (
	"errors"
	"fmt"
	"github.com/nyudlts/go-aspace"
	"io"
	"os"
	"path/filepath"
)

type ResourceInfo struct {
	RepoID     int
	RepoSlug   string
	ResourceID int
}

var client *aspace.ASClient

func CreateAspaceClient(config string, environment string, timeout int) error {
	var err error
	client, err = aspace.NewClient(config, environment, timeout)
	if err != nil {
		return err
	}
	return nil
}

// check the application flags
func CheckFlags(config string, environment string, format string, resource int, repository int) error {
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

func GetRepositoryMap(repository int, environment string) (map[string]int, error) {
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

func GetResourceIDs(repMap map[string]int, resource int) ([]ResourceInfo, error) {

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

func CreateWorkDirectory(workDirPath string) error {
	//determine if the directory already exists or if there is an error, if so return an error
	if _, err := os.Stat(workDirPath); err == nil {
		return fmt.Errorf("work directory %s already exists")
	} else if errors.Is(err, os.ErrNotExist) {
		//the workDir doesn't exist -- create it if there are no other errors
	} else {
		return err
	}

	err := os.Mkdir(workDirPath, 0755)
	if err != nil {
		return err
	}

	return nil
}

func CreateExportDirectories(workDirPath string, repositoryMap map[string]int, unpublishedResources bool) error {
	for slug := range repositoryMap {

		repositoryDir := filepath.Join(workDirPath, slug)
		exportDir := filepath.Join(repositoryDir, "exports")
		failureDir := filepath.Join(repositoryDir, "invalid")
		unpublishedDir := filepath.Join(repositoryDir, "unpublished")

		if _, err := os.Stat(repositoryDir); os.IsNotExist(err) {
			innerErr := os.Mkdir(repositoryDir, 0755)
			if innerErr != nil {
				return innerErr
			} else {
				PrintAndLog(fmt.Sprintf("created repository directory %s", repositoryDir), INFO)
			}
		}

		//create the repository export directory
		if _, err := os.Stat(exportDir); os.IsNotExist(err) {
			innerErr := os.Mkdir(exportDir, 0755)
			if innerErr != nil {
				return innerErr
			} else {
				PrintAndLog(fmt.Sprintf("created exports directory %s", exportDir), INFO)
			}
		}

		//create the repository failure directory
		if _, err := os.Stat(failureDir); os.IsNotExist(err) {
			innerErr := os.Mkdir(failureDir, 0755)
			if innerErr != nil {
				return innerErr
			} else {
				PrintAndLog(fmt.Sprintf("created failures directory %s", failureDir), INFO)
			}
		}

		if unpublishedResources == true {
			if _, err := os.Stat(unpublishedDir); os.IsNotExist(err) {
				innerErr := os.Mkdir(unpublishedDir, 0755)
				if innerErr != nil {
					return innerErr
				} else {
					PrintAndLog(fmt.Sprintf("created unpublished directory %s", unpublishedDir), INFO)
				}
			}
		}
	}

	return nil
}

func Cleanup(workDir string) error {
	//remove any empty directories
	err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			} else {
				defer f.Close()
				_, err = f.Readdirnames(1)
				if err == io.EOF {
					PrintAndLog(fmt.Sprintf("removing empty directory at: %s", path), INFO)
					innerErr := os.Remove(path)
					if innerErr != nil {
						return innerErr
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	//move the logfile to the workdir
	newLoc := filepath.Join(workDir, logfile)
	err = os.Rename(logfile, newLoc)
	if err != nil {
		return err
	}
	PrintAndLog(fmt.Sprintf("moved log file to %s", newLoc), INFO)

	//move the reportFile to the workdir
	newLoc = filepath.Join(workDir, reportFile)
	err = os.Rename(reportFile, newLoc)
	if err != nil {
		return err
	}
	PrintAndLog(fmt.Sprintf("moved report file to %s", newLoc), INFO)

	return nil
}
