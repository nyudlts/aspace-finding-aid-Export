package main

import (
	"fmt"
	"github.com/nyudlts/go-aspace"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ExportResult struct {
	Status string
	URI    string
	Error  string
}

var numSkipped = 0
var numValidationErr = 0

func exportResources() {
	resourceChunks := chunkResources()
	resultChannel := make(chan []ExportResult)

	for i, chunk := range resourceChunks {
		go exportChunk(chunk, resultChannel, i+1)
	}

	var results []ExportResult

	for range resourceChunks {
		chunk := <-resultChannel
		log.Println("INFO Adding", len(chunk), "uris to uri list")
		results = append(results, chunk...)
	}

	fmt.Printf("\n%d resources proccessed:\n", len(results))
	fmt.Printf("  * %d resources skipped\n", numSkipped)
	fmt.Printf("  * %d validation errors\n", numValidationErr)

	//print any errors encountered to terminal
	errors := []ExportResult{}
	for _, result := range results {
		if result.Status == "ERROR" {
			errors = append(errors, result)
		}
	}

	if len(errors) > 0 {
		fmt.Println("Errors Encountered:")
		for _, e := range errors {
			fmt.Println("      ", e)
		}
	} else {
		fmt.Println("  * No errors encountered during processing")
	}

}
func exportChunk(resourceInfoChunk []ResourceInfo, resultChannel chan []ExportResult, workerID int) {
	fmt.Println("  * Starting worker", workerID, "processing", len(resourceInfoChunk), "resources")
	log.Println("INFO Starting worker", workerID, "processing", len(resourceInfoChunk), "resources")
	var results = []ExportResult{}

	//loop through the chunk
	for _, rInfo := range resourceInfoChunk {

		//get the resource object
		resource, err := client.GetResource(rInfo.RepoID, rInfo.ResourceID)
		if err != nil {
			results = append(results, ExportResult{Status: "ERROR", URI: "", Error: err.Error()})
			log.Printf("INFO worker %d could not retrieve resource %s", workerID, resource.URI)
			continue
		}

		//check if the resource is set to be published
		if unpublished == false && resource.Publish != true {
			log.Printf("INFO worker %d resource %s not set to publish, skipping", workerID, resource.URI)
			numSkipped = numSkipped + 1
			results = append(results, ExportResult{Status: "SKIPPED", URI: resource.URI, Error: ""})
			continue
		}

		if marc == true {
			//export the marc record
			results = append(results, exportMarc(rInfo, resource, workerID))
		} else {

		}
	}
	resultChannel <- results
}

func exportMarc(info ResourceInfo, res aspace.Resource, workerID int) ExportResult {
	endpoint := fmt.Sprintf("/repositories/%d/resources/marc21/%d.xml", info.RepoID, info.ResourceID)

	//get the marc record
	marcBytes, err := client.GetEndpoint(endpoint)
	if err != nil {
		log.Printf("INFO worker %d could not retrieve resource %s", workerID, res.URI)
		return ExportResult{Status: "ERROR", URI: res.URI, Error: err.Error()}
	}

	//create the output filename
	t := time.Now()
	tf := t.Format("20060102")
	marcFilename := strings.ToLower(MergeIDs(res) + "_" + tf + ".xml")

	//set the location to write the marc record
	var marcPath string
	if unpublished == true && res.Publish == false {
		marcPath = filepath.Join(workDir, info.RepoSlug, "unpublished", marcFilename)
	} else {
		marcPath = filepath.Join(workDir, info.RepoSlug, "exports", marcFilename)
	}

	//write the marc file
	err = ioutil.WriteFile(marcPath, marcBytes, 0777)
	if err != nil {
		log.Printf("INFO worker %d could not write the marc record %s", workerID, res.URI)
		return ExportResult{Status: "ERROR", URI: "", Error: err.Error()}
	}

	//return the result
	log.Printf("INFO worker %d exported resource %s - %s", workerID, res.URI, res.EADID)
	return ExportResult{Status: "SUCCESS", URI: res.URI, Error: ""}
}

func exportEAD(info ResourceInfo, res aspace.Resource, workerID int) ExportResult {

	//get the ead as bytes
	eadBytes, err := client.GetEADAsByteArray(info.RepoID, info.ResourceID)
	if err != nil {
		log.Printf("INFO worker %d could not retrieve resource %s", workerID, res.URI)
		return ExportResult{Status: "ERROR", URI: res.URI, Error: err.Error()}
	}

	//create the output filename
	faFilename := res.EADID + ".xml"
	outputFile := filepath.Join(workDir, info.RepoSlug, "exports", faFilename)

	//validate the output
	if validate == true {
		err = aspace.ValidateEAD(eadBytes)
		if err != nil {
			numValidationErr = numValidationErr + 1
			log.Printf("WARNING worker %d resource %s - %s failed validation, writing to failures directory", workerID, res.URI, res.EADID)
			outputFile = filepath.Join(workDir, info.RepoSlug, "failures", faFilename)
		}
	}

	//create the output file
	err = ioutil.WriteFile(outputFile, eadBytes, 0777)
	if err != nil {
		log.Printf("INFO worker %d could not write the ead file %s", workerID, res.URI)
		return ExportResult{Status: "ERROR", URI: "", Error: err.Error()}
	}

	//return the result
	log.Printf("INFO worker %d exported resource %s - %s", workerID, res.URI, res.EADID)
	return ExportResult{Status: "SUCCESS", URI: res.URI, Error: ""}
}

func chunkResources() [][]ResourceInfo {
	var divided [][]ResourceInfo

	chunkSize := (len(resourceInfo) + workers - 1) / workers

	for i := 0; i < len(resourceInfo); i += chunkSize {
		end := i + chunkSize

		if end > len(resourceInfo) {
			end = len(resourceInfo)
		}

		divided = append(divided, resourceInfo[i:end])
	}
	return divided
}

func tabReformatXML(path string) error {

	//lint the ead file
	reformattedBytes, err := exec.Command("xmllint", "--format", path).Output()
	if err != nil {
		return fmt.Errorf("could not reformat %s", path)
	}

	//delete the original
	err = os.Remove(path)
	if err != nil {
		return fmt.Errorf("could not delete %s", path)
	}

	//rewrite the file
	err = ioutil.WriteFile(path, reformattedBytes, 0644)
	if err != nil {
		return fmt.Errorf("could not write reformated bytes to %s", path)
	}

	return nil
}

func MergeIDs(r aspace.Resource) string {
	ids := r.ID0
	for _, i := range []string{r.ID1, r.ID2, r.ID3} {
		if i != "" {
			ids = ids + "_" + i
		}
	}
	return ids
}
