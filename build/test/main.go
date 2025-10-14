package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func FindFilesInARepoBySHA(username, password, server, repository string, sha256s []string) ([]byte, error) {
	if len(sha256s) == 0 {
		return nil, fmt.Errorf("SHA list cannot be empty")
	}

	// Build SHA256 conditions for the AQL query
	var shaConditions []string
	for _, sha := range sha256s {
		shaConditions = append(shaConditions, fmt.Sprintf(`{"sha256": "%s"}`, strings.TrimSpace(sha)))
	}

	var aqlQuery string

	shaOrConditions := strings.Join(shaConditions, ", ")
	aqlQueryTemplate := `items.find(
    {
        "$and": [
            {"repo": "%s"},
            {"$or": [%s]}
        ]
    }
).include("path","sha256","actual_sha1","actual_md5")`

	aqlQuery = fmt.Sprintf(aqlQueryTemplate, repository, shaOrConditions)
	payload := bytes.NewBuffer([]byte(aqlQuery))
	aqlURL := server + "/artifactory/api/search/aql"

	req, err := http.NewRequest("POST", aqlURL, payload)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.SetBasicAuth(username, password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to Artifactory: %w", err)
	}
	defer resp.Body.Close()

	// 6. Read the full response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// 7. Check the HTTP status code for non-successful responses
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Artifactory API failed with status %s. Response body: %s", resp.Status, string(bodyBytes))
	}

	return bodyBytes, nil
}

func main() {
	response, err := FindFilesInARepoBySHA("<user>", "<password>", "https://ecosysjfrog.jfrog.io", "docker-local", []string{"661ff4d9561e3fd050929ee5097067c34bafc523ee60f5294a37fd08056a73ca", "f256aee169df8b3e2eb2cee4840048e9f05837771f5d91799eec15693d27f55b"})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Response from Artifactory:")
	fmt.Println(string(response))
}
