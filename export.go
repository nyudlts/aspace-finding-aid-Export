package main

import (
	"fmt"
	"github.com/nyudlts/go-aspace"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
		if marc == true {
			go exportMARCChunk(chunk, resultChannel, i+1)
		} else {
			go exportFindingAidChunk(chunk, resultChannel, i+1)
		}
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

func exportMARCChunk(resourceInfoChunk []ResourceInfo, resultChannel chan []ExportResult, workerID int) {
	fmt.Println("  * Starting worker", workerID, "processing", len(resourceInfoChunk), "resources")
	log.Println("INFO Starting worker", workerID, "processing", len(resourceInfoChunk), "resources")
	var results = []ExportResult{}

	for _, rInfo := range resourceInfoChunk {
		//get the resource object

		resource, err := client.GetResource(rInfo.RepoID, rInfo.ResourceID)
		if err != nil {
			results = append(results, ExportResult{Status: "ERROR", URI: "", Error: err.Error()})
			continue
		}

		if unpublished == false && resource.Publish != true {
			log.Printf("INFO worker %d resource %s not set to publish, skipping", workerID, resource.URI)
			numSkipped = numSkipped + 1
			results = append(results, ExportResult{Status: "SUCCESS", URI: resource.URI, Error: ""})
			continue
		}

		endpoint := fmt.Sprintf("/repositories/%d/resources/marc21/%d.xml", rInfo.RepoID, rInfo.ResourceID)

		marcBytes, err := client.GetEndpoint(endpoint)
		if err != nil {
			results = append(results, ExportResult{Status: "ERROR", URI: resource.URI, Error: err.Error()})
			continue
		}

		//create the output filename
		t := time.Now()
		tf := t.Format("20060102")
		marcFilename := resource.EADID + "_" + tf + ".xml"
		var marcPath string

		if unpublished == true {
			marcPath = filepath.Join(workDir, rInfo.RepoSlug, "unpublished", marcFilename)
		} else {
			marcPath = filepath.Join(workDir, rInfo.RepoSlug, "exports", marcFilename)
		}

		err = ioutil.WriteFile(marcPath, marcBytes, 0777)
		if err != nil {
			results = append(results, ExportResult{Status: "ERROR", URI: "", Error: err.Error()})
			continue
		}

		log.Printf("INFO worker %d exported resource %s - %s", workerID, resource.URI, resource.EADID)
		results = append(results, ExportResult{Status: "SUCCESS", URI: resource.URI, Error: ""})
	}
	resultChannel <- results
}

func exportFindingAidChunk(resourceInfoChunk []ResourceInfo, resultChannel chan []ExportResult, workerID int) {
	fmt.Println("  * Starting worker", workerID, "processing", len(resourceInfoChunk), "resources")
	log.Println("INFO Starting worker", workerID, "processing", len(resourceInfoChunk), "resources")

	var results = []ExportResult{}
	for _, rInfo := range resourceInfoChunk {
		//get the resource object
		resource, err := client.GetResource(rInfo.RepoID, rInfo.ResourceID)
		if err != nil {
			results = append(results, ExportResult{Status: "ERROR", URI: resource.URI, Error: err.Error()})
			continue
		}

		//skip anything not set to publish
		if resource.Publish != true {
			log.Printf("INFO worker %d resource %s not set to publish, skipping", workerID, resource.URI)
			numSkipped = numSkipped + 1
			results = append(results, ExportResult{Status: "SUCCESS", URI: resource.URI, Error: ""})
			continue
		}

		//skip anything with a blank eadid
		if resource.EADID == "" {
			log.Printf("ERROR worker %d: resource %s had a blank EADID", workerID, resource.URI)
			numSkipped = numSkipped + 1
			results = append(results, ExportResult{Status: "ERROR", URI: resource.URI, Error: "Resource had a blank EADID, skipping"})
			continue
		}

		//get the ead as bytes
		eadBytes, err := client.GetEADAsByteArray(rInfo.RepoID, rInfo.ResourceID)
		if err != nil {
			results = append(results, ExportResult{Status: "ERROR", URI: resource.URI, Error: err.Error()})
			continue
		}

		//create the output filename
		faFilename := resource.EADID + ".xml"
		outputFile := filepath.Join(workDir, rInfo.RepoSlug, "exports", faFilename)

		//validate the output
		if validate == true {
			err = aspace.ValidateEAD(eadBytes)
			if err != nil {
				numValidationErr = numValidationErr + 1
				log.Printf("ERROR worker %d resource %s - %s failed validation, writing to failures directory", workerID, resource.URI, resource.EADID)
				outputFile = filepath.Join(workDir, rInfo.RepoSlug, "failures", faFilename)
			}
		}

		//create the output file
		eadFile, err := os.OpenFile(outputFile, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			results = append(results, ExportResult{Status: "ERROR", URI: resource.URI, Error: err.Error()})
			log.Printf("ERROR worker %d could not create file %s", workerID, faFilename)
			continue
		}
		defer eadFile.Close()

		//write the ead to file
		_, err = eadFile.Write(eadBytes)
		if err != nil {
			results = append(results, ExportResult{Status: "ERROR", URI: resource.URI, Error: err.Error()})
			log.Printf("ERROR worker %d could not write to file %s", workerID, faFilename)
			continue
		}

		//reformat the ead with tabs
		if reformat == true {
			err = tabReformatXML(outputFile)
			if err != nil {
				log.Printf("ERROR worker %d could not reformat %s", workerID, outputFile)
			}
		}

		//everything worked.
		log.Printf("INFO worker %d exported resource %s - %s", workerID, resource.URI, resource.EADID)
		results = append(results, ExportResult{Status: "SUCCESS", URI: resource.URI, Error: ""})
	}

	fmt.Printf("  * Worker %d finished, processed %d resources\n", workerID, len(resourceInfoChunk))
	resultChannel <- results
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
