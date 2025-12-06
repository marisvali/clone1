//go:build http_disabled

package main

import (
	"github.com/google/uuid"
)

func InitializeIdInDbHttp(user string,
	releaseVersion int64,
	simulationVersion int64,
	inputVersion int64,
	id uuid.UUID) {
}

func UploadDataToDbHttp(user string,
	releaseVersion int64,
	simulationVersion int64,
	inputVersion int64,
	id uuid.UUID, data []byte) {
}

func SetUserDataHttp(user string, data string) {
}

func GetUserDataHttp(user string) string {
	return ""
}
