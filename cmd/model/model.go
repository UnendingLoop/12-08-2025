package model

import (
	"errors"
	"sync"
	"sync/atomic"
)

type Status string

const ( //статусы сразу и для тасков, и для файлов
	StatusPending    Status = "pending"
	StatusInProgress Status = "in progress"
	StatusReady      Status = "ready"
	StatusError      Status = "error"
)

type Task struct {
	TID        string     `json:"id"`
	FilesCount int        `json:"files_count"`
	Files      []FileInfo `json:"files"`
	Status     Status     `json:"task_status"`
	Archive    string     `json:"archive_link"`
}

type FileInfo struct {
	URL    string `json:"file_URL"`
	Name   string `json:"-"`           //фактическое имя файла
	Status Status `json:"file_status"` //pending, ready, error
	Error  string `json:"-"`
}

type TasksMap struct {
	Mapa             map[string]*Task
	ActiveTasksCount atomic.Int32 //для отслеживания загруженности
	sync.RWMutex                  //только для операций с картой
	Channel          chan *Task
}

var (
	ErrFailedToZIP    = errors.New("failed to add files to archive")
	ErrFileFormat     = errors.New("invalid filetype: only .pdf and .jpeg are acceptable")
	ErrDownloadFailed = errors.New("failed to download file")
	ErrInvalidLink    = errors.New("invalid download-link format")
	ErrTaskIsFull     = errors.New("the task aleady contains max number of files")
	ErrBusy           = errors.New("already processing max number of tasks. Try again later")
)
