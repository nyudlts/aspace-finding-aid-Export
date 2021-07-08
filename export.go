package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type ExportResult struct {
	Status string
	URI string
	Error string
}

func exportResources() {
	resourceChunks := chunkResources()
	resultChannel := make(chan []ExportResult)
	for i, chunk := range resourceChunks {
		go exportFindingAidChunk(chunk, resultChannel, i+1)
	}

	var results []ExportResult

	for range resourceChunks {
		chunk := <-resultChannel
		log.Println("INFO\tAdding", len(chunk), "uris to uri list")
		results = append(results, chunk...)
	}

	//print any errors encountered to terminal
	for _, result := range results {
		if result.Status == "ERROR" {
			fmt.Println(result)
		}
	}

}

func exportFindingAidChunk(resourceInfoChunk []ResourceInfo, resultChannel chan []ExportResult, workerID int) {
	fmt.Println("  * Starting worker", workerID, "processing", len(resourceInfoChunk), "resources")
	log.Println("INFO\tStarting worker", workerID, "processing", len(resourceInfoChunk), "resources")
	results := []ExportResult{}
	for _, rInfo := range resourceInfoChunk {
		//get the resource object
		resource, err := client.GetResource(rInfo.RepoID, rInfo.ResourceID)
		if err != nil {
			results = append(results, ExportResult{ Status: "ERROR", URI:    "", Error:  err.Error() })
			continue
		}

		//skip anything not set to publish
		if resource.Publish != true {
			log.Printf("INFO\tworker %d resource %s not set to publish, skipping", workerID, resource.URI)
			results = append(results, ExportResult{ Status: "SUCCESS", URI: resource.URI, Error:  "" })
			continue
		}

		//skip anything with a blank eadid
		if resource.EADID == "" {
			log.Printf("ERROR\tworker %d: resource %s had a blank EADID", workerID, resource.URI)
			results = append(results, ExportResult{ Status: "ERROR", URI: resource.URI, Error:  "Resource had a blank EADID, skipping" })
			continue
		}

		//get the ead as bytes
		eadBytes, err := client.GetEADAsByteArray(rInfo.RepoID, rInfo.ResourceID)
		if err != nil {
			results = append(results, ExportResult{ Status: "ERROR", URI: resource.URI, Error:  err.Error() })
			continue
		}

		//create the output file
		faFilename := resource.EADID + ".xml"
		outputFile := filepath.Join(workDir, rInfo.RepoSlug, "exports", faFilename)
		f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_RDWR, 0755)
		if err != nil {
			results = append(results, ExportResult{ Status: "ERROR", URI: resource.URI, Error:  err.Error() })
			continue
		}
		defer f.Close()

		//write the bytes to the output file
		_, err = f.Write(eadBytes)
		if err != nil {
			results = append(results, ExportResult{ Status: "ERROR", URI: resource.URI, Error:  err.Error() })
			continue
		}

		//everything worked.
		log.Printf("INFO\tworker %d exported resource %s", workerID, resource.URI)
		results = append(results, ExportResult{ Status: "SUCCESS", URI:    resource.URI, Error:  "" })
	}

	fmt.Printf("  * worker %d finished, processed %d resources\n", workerID, len(resourceInfoChunk))
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