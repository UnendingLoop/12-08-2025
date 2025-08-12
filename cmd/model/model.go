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
	Files      []*FileInfo  `json:"files"`
	Status     Status       `json:"task_status"`
	Archive    *string      `json:"archive_URI,omitempty"`
	sync.RWMutex
	TmpDir  string `json:"-"`
	ArchDir string `json:"-"`
}

type FileInfo struct {
	URL    string `json:"file_URL"`
	Name   string `json:"-"`           //фактическое имя файла
	Status Status `json:"file_status"` //pending, ready, error
	Error  *error `json:"-"`
}

type TasksMap struct {
	Mapa             map[string]*Task
	ActiveTasksCount atomic.Int32 //для отслеживания загруженности, макс 3
	Channel          chan *Task   //сюда отправлять экземпляры задач
	sync.RWMutex
	Done     chan struct{} // сигнал на завершение
	ValidExt []string
	TmpDir   string
	ArchDir  string
}
type NewLink struct {
	URL string `json:"file_URL"`
}
type Config struct {
	ValidExt []string
	AppPort  int
	TmpDir   string
	ArchDir  string
}

var (
	ErrFailedToZIP    = errors.New("failed to add files to archive")
	ErrFileFormat     = errors.New("invalid filetype: only 'pdf' and 'jpeg' are acceptable")
	ErrDownloadFailed = errors.New("failed to download file")
	ErrInvalidLink    = errors.New("invalid download-link format")
	ErrTaskIsFull     = errors.New("the task aleady contains max number of files")
	ErrBusy           = errors.New("already processing max number of tasks. Try again later")
)
