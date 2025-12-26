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
// body of the response as a string. It returns an error if the call to the
// server fails. Other errors are considered programmer errors and cause a
// panic.
func makeHttpRequest(
	url string,
	fields map[string]string,
	files map[string][]byte,
) (string, error) {
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
	if err != nil {
		return "", err
	}
	if response.StatusCode != 200 {
		return "", fmt.Errorf("http request failed: %d", response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	Check(err)
	return string(data), nil
}

func InitializeIdInDbHttp(user string,
	releaseVersion int64,
	simulationVersion int64,
	inputVersion int64,
	id uuid.UUID) error {
	url := "https://playful-patterns.com/submit-playthrough-clone1.php"
	_, err := makeHttpRequest(url,
		map[string]string{
			"user":               user,
			"release_version":    strconv.FormatInt(releaseVersion, 10),
			"simulation_version": strconv.FormatInt(simulationVersion, 10),
			"input_version":      strconv.FormatInt(inputVersion, 10),
			"id":                 id.String()},
		map[string][]byte{})
	return err
}

func UploadDataToDbHttp(user string,
	releaseVersion int64,
	simulationVersion int64,
	inputVersion int64,
	id uuid.UUID, data []byte) error {
	url := "https://playful-patterns.com/submit-playthrough-clone1.php"
	_, err := makeHttpRequest(url,
		map[string]string{
			"user":               user,
			"release_version":    strconv.FormatInt(releaseVersion, 10),
			"simulation_version": strconv.FormatInt(simulationVersion, 10),
			"input_version":      strconv.FormatInt(inputVersion, 10),
			"id":                 id.String()},
		map[string][]byte{"playthrough": data})
	return err
}

func SetUserDataHttp(user string, data string) error {
	url := "https://playful-patterns.com/set-user-data-clone1.php"
	_, err := makeHttpRequest(url,
		map[string]string{"user": user, "data": data},
		map[string][]byte{})
	return err
}

func GetUserDataHttp(user string) (string, error) {
	url := "https://playful-patterns.com/get-user-data-clone1.php"
	return makeHttpRequest(url,
		map[string]string{"user": user},
		map[string][]byte{})
}

func LogErrorHttp(user string,
	releaseVersion int64,
	simulationVersion int64,
	inputVersion int64,
	id uuid.UUID,
	errorMsg string,
	data []byte) error {
	url := "https://playful-patterns.com/log-error-clone1.php"
	_, err := makeHttpRequest(url,
		map[string]string{
			"user":               user,
			"release_version":    strconv.FormatInt(releaseVersion, 10),
			"simulation_version": strconv.FormatInt(simulationVersion, 10),
			"input_version":      strconv.FormatInt(inputVersion, 10),
			"id":                 id.String(),
			"error":              errorMsg,
		},
		map[string][]byte{"playthrough": data})
	return err
}
