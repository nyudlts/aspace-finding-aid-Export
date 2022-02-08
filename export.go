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

	//seperate result types
	successes := []ExportResult{}
	errors := []ExportResult{}
	warnings := []ExportResult{}
	skipped := []ExportResult{}

	for _, result := range results {
		switch result.Status {
		case "SUCCESS":
			successes = append(successes, result)
		case "ERROR":
			errors = append(errors, result)
		case "WARNING":
			warnings = append(warnings, result)
		case "SKIPPED":
			warnings = append(skipped, result)
		default:
		}
	}

	//reporting
	fmt.Printf("\n%d Resources proccessed:\n", len(results))
	fmt.Printf("  %d Successful exports\n", len(successes))
	fmt.Printf("  %d Skipped resources\n", len(skipped))

	fmt.Printf("  %d Exports with warnings\n", len(warnings))
	if len(warnings) > 0 {
		for _, w := range warnings {
			w.Error = strings.ReplaceAll(w.Error, "\n", " ")
			fmt.Println("    ", w)
		}
	}

	fmt.Printf("  %d Errors Encountered\n", len(errors))
	if len(errors) > 0 {
		for _, e := range errors {
			e.Error = strings.ReplaceAll(e.Error, "\n", " ")
			fmt.Println("    ", e)
		}
	}
}

func exportChunk(resourceInfoChunk []ResourceInfo, resultChannel chan []ExportResult, workerID int) {
	fmt.Println("  Starting worker", workerID, "processing", len(resourceInfoChunk), "resources")
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
			results = append(results, exportEAD(rInfo, resource, workerID))
		}
	}
	resultChannel <- results
}

func exportMarc(info ResourceInfo, res aspace.Resource, workerID int) ExportResult {

	//get the marc record
	marcBytes, err := client.GetMARCAsByteArray(info.RepoID, info.ResourceID)
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

	//validate the output
	warning := false
	var warningType = ""
	if validate == true {
		err = aspace.ValidateMARC(marcBytes)
		if err != nil {
			warning = true
			fmt.Println(err.Error())
			warningType = "failed MARC21 validation, writing to failures directory"
			log.Printf("WARNING worker %d resource %s - %s %s %s", workerID, res.URI, res.EADID, warningType, err.Error())
			marcPath = filepath.Join(workDir, info.RepoSlug, "failures", marcFilename)
		}
	}

	//write the marc file
	err = ioutil.WriteFile(marcPath, marcBytes, 0777)
	if err != nil {
		log.Printf("INFO worker %d could not write the marc record %s", workerID, res.URI)
		return ExportResult{Status: "ERROR", URI: "", Error: err.Error()}
	}

	//return the result
	if warning == true {
		log.Printf("INFO worker %d exported resource %s - %s with warning", workerID, res.URI, marcFilename)
		return ExportResult{Status: "WARNING", URI: res.URI, Error: warningType}
	}
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
	eadFilename := strings.ToLower(MergeIDs(res) + ".xml")
	outputFile := filepath.Join(workDir, info.RepoSlug, "exports", eadFilename)

	//validate the output
	warning := false
	var warningType = ""
	if validate == true {
		err = aspace.ValidateEAD(eadBytes)
		if err != nil {
			warning = true
			warningType = "failed EAD2002 validation, writing to failures directory"
			log.Printf("WARNING worker %d resource %s - %s %s", workerID, res.URI, res.EADID, warningType)
			outputFile = filepath.Join(workDir, info.RepoSlug, "failures", eadFilename)
		}
	}

	//create the output file
	err = ioutil.WriteFile(outputFile, eadBytes, 0777)
	if err != nil {
		log.Printf("INFO worker %d could not write the ead file %s", workerID, res.URI)
		return ExportResult{Status: "ERROR", URI: "", Error: err.Error()}
	}

	//reformat the ead with tabs
	if reformat == true {
		err = tabReformatXML(outputFile)
		if err != nil {
			log.Printf("WARNING worker %d could not reformat %s", workerID, outputFile)
		}
	}

	//return the result

	if warning == true {
		log.Printf("INFO worker %d exported resource %s - %s with warning", workerID, res.URI, eadFilename)
		return ExportResult{Status: "WARNING", URI: res.URI, Error: warningType}
	}
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
