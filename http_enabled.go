//go:build http_enabled

package main

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
)

// makeHttpRequest makes a POST HTTP request to an endpoint and returns the
// body of the response as a string.
func makeHttpRequest(url string, fields map[string]string, files map[string][]byte) string {
	// Create a buffer to write our multipart form data.
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	for k, v := range fields {
		err := writer.WriteField(k, v)
		Check(err)
	}
	for k, v := range files {
		part, err := writer.CreateFormFile(k, k)
		Check(err)
		_, err = part.Write(v)
		Check(err)
	}
	err := writer.Close()
	Check(err)

	// Create a POST request with the multipart form data.
	request, err := http.NewRequest("POST", url, &requestBody)
	Check(err)
	request.Header.Set("content-type", writer.FormDataContentType())

	// Perform the request.
	client := &http.Client{}
	response, err := client.Do(request)
	Check(err)
	if response.StatusCode != 200 {
		Check(fmt.Errorf("http request failed: %d", response.StatusCode))
	}
	data, err := io.ReadAll(response.Body)
	Check(err)
	return string(data)
}

func InitializeIdInDbHttp(user string,
	releaseVersion int64,
	simulationVersion int64,
	inputVersion int64,
	id uuid.UUID) {
	url := "https://playful-patterns.com/submit-playthrough-clone1.php"
	makeHttpRequest(url,
		map[string]string{
			"user":               user,
			"release_version":    strconv.FormatInt(releaseVersion, 10),
			"simulation_version": strconv.FormatInt(simulationVersion, 10),
			"input_version":      strconv.FormatInt(inputVersion, 10),
			"id":                 id.String()},
		map[string][]byte{})
}

func UploadDataToDbHttp(user string,
	releaseVersion int64,
	simulationVersion int64,
	inputVersion int64,
	id uuid.UUID, data []byte) {
	url := "https://playful-patterns.com/submit-playthrough-clone1.php"
	makeHttpRequest(url,
		map[string]string{
			"user":               user,
			"release_version":    strconv.FormatInt(releaseVersion, 10),
			"simulation_version": strconv.FormatInt(simulationVersion, 10),
			"input_version":      strconv.FormatInt(inputVersion, 10),
			"id":                 id.String()},
		map[string][]byte{"playthrough": data})
}

func SetUserDataHttp(user string, data string) {
	url := "https://playful-patterns.com/set-user-data-clone1.php"
	makeHttpRequest(url,
		map[string]string{"user": user, "data": data},
		map[string][]byte{})
}

func GetUserDataHttp(user string) string {
	url := "https://playful-patterns.com/get-user-data-clone1.php"
	return makeHttpRequest(url,
		map[string]string{"user": user},
		map[string][]byte{})
}
