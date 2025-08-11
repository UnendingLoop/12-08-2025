package model

import (
	"errors"
	"sync"
	"sync/atomic"
)

type Status string //статусы сразу и для тасков, и для файлов

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in progress"
	StatusReady      Status = "ready"
	StatusError      Status = "error"
)

type Task struct {
	TID        string       `json:"id"`
	FilesCount atomic.Int32 `json:"-"`
	Files      []*FileInfo  `json:"files_info"`
	Status     Status       `json:"task_status"`
	Archive    *string      `json:"archive_link,omitempty"`
	sync.RWMutex
}

type FileInfo struct {
	URL    string `json:"file_URL"`
	Name   string `json:"-"`           //фактическое имя файла
	Status Status `json:"file_status"` //pending, ready, error
	Error  *error `json:"-"`
}

type TasksMap struct {
	Mapa             map[string]*Task
	ActiveTasksCount atomic.Int32 //для отслеживания загруженности
	Channel          chan *Task
	sync.RWMutex                   //только для доступа к мапе
	Done             chan struct{} // сигнал на завершение
}
type NewLink struct {
	URL string `json:"file_URL"`
}

var (
	ErrFailedToZIP    = errors.New("failed to add files to archive")
	ErrFileFormat     = errors.New("invalid filetype: only 'pdf' and 'jpeg' are acceptable")
	ErrDownloadFailed = errors.New("failed to download file")
	ErrInvalidLink    = errors.New("invalid download-link format")
	ErrTaskIsFull     = errors.New("the task aleady contains max number of files")
	ErrBusy           = errors.New("already processing max number of tasks. Try again later")

	ValidExt = []string{".pdf", ".jpeg", ".jpg"}
)
