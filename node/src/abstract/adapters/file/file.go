package file

import (
	"archive/tar"
	"mime/multipart"
)

type IFile interface {
	CheckFileFromStorage(storageRoot string, topicId string, key string) bool
	SaveFileToStorage(storageRoot string, fh *multipart.FileHeader, topicId string, key string) error
	SaveDataToStorage(storageRoot string, data []byte, topicId string, key string) error
	SaveTarFileItemToStorage(storageRoot string, fh *tar.Reader, topicId string, key string) error
	ReadFileFromStorage(storageRoot string, topicId string, key string) ([]byte, error)
	CheckFileFromGlobalStorage(storageRoot string, key string) bool
	ReadFileFromGlobalStorage(storageRoot string, key string) (string, error)
	SaveFileToGlobalStorage(storageRoot string, fh *multipart.FileHeader, key string, overwrite bool) error
	SaveDataToGlobalStorage(storageRoot string, data []byte, key string, overwrite bool) error
}
