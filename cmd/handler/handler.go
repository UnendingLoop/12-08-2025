package handler

import (
	"12-08-2025/cmd/model"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// TasksHandler contains map of tasks with mutex, provides access to HTTP-handlers
type TasksHandler struct {
	Pool *model.TasksMap
}

// CreateNewTask - creates a task in model.TasksMap.Mapa if there are less than 3 tasks in progress
func (h *TasksHandler) CreateNewTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Pool.ActiveTasksCount.Load() == 3 {
		http.Error(w, fmt.Sprintf("Failed to create new task: %v.", model.ErrBusy), http.StatusServiceUnavailable)
		return
	}

	h.Pool.Lock()
	newID := uuid.New().String()
	newTask := &model.Task{
		TID:     newID,
		Files:   []*model.FileInfo{},
		Status:  model.StatusPending,
		ArchDir: h.Pool.ArchDir,
		TmpDir:  h.Pool.TmpDir}

	h.Pool.Mapa[newID] = newTask
	h.Pool.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newTask); err != nil {
		http.Error(w, "Failed to encode task info. Try again.", http.StatusInternalServerError)
		return
	}
}

// AddLinkToTask checks if such task exists by ID from path, creates model.FileInfo and adds to task. If there are 3 files in task - sends it to chan
func (h *TasksHandler) AddLinkToTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	tid := chi.URLParam(r, "id")
	if tid == "" {
		http.Error(w, "Empty task ID", http.StatusBadRequest)
		return
	}

	h.Pool.RLock()
	task, exists := h.Pool.Mapa[tid]
	h.Pool.RUnlock()

	task.Lock()
	if !exists {
		http.Error(w, fmt.Sprintf("Task with ID '%s' doesn't exist", tid), http.StatusNotFound)
		return
	}

	if task.FilesCount.Load() == 3 {
		http.Error(w, fmt.Sprintf("Failed to add link to task '%s': %v", tid, model.ErrTaskIsFull), http.StatusConflict)
		return
	}
	newLink := model.NewLink{}
	if err := json.NewDecoder(r.Body).Decode(&newLink); err != nil { //должен считаться только URL
		http.Error(w, fmt.Sprintf("Failed to decode task '%s' file link", tid), http.StatusBadRequest)
		return
	}
	if newLink.URL == "" {
		http.Error(w, "Specified URL is empty", http.StatusBadRequest)
		return
	}
	//чистка от пробелов + валидация по фортмау
	newLink.URL = strings.TrimSpace(newLink.URL)
	if _, err := url.ParseRequestURI(newLink.URL); err != nil {
		http.Error(w, fmt.Sprintf("Invalid URL format '%s'", newLink.URL), http.StatusBadRequest)
		return
	}
	//проверка расширения файла
	found := false
	h.Pool.RLock()
	for _, v := range h.Pool.ValidExt {
		if strings.HasSuffix(newLink.URL, v) {
			found = true
			break
		}
	}
	h.Pool.RUnlock()

	if !found {
		http.Error(w, fmt.Sprintf("Failed to add link: %v", model.ErrFileFormat), http.StatusBadRequest)
		return
	}

	//валидация ссылки - проверка на уникальность внутри одной таски
	for _, v := range task.Files {
		if v.URL == newLink.URL {
			http.Error(w, fmt.Sprintf("Link '%s' already in task '%s'", newLink.URL, tid), http.StatusBadRequest)
			task.Unlock()
			return
		}
	}
	newFile := model.FileInfo{URL: newLink.URL}

	newFile.Status = model.StatusPending
	task.Files = append(task.Files, &newFile)
	task.Unlock()

	task.FilesCount.Add(1)

	w.WriteHeader(http.StatusNoContent)

	//проверка на кол-во файлов: если 3 - передаем в канал для обработки воркерами
	if task.FilesCount.Load() == 3 {
		select {
		case <-h.Pool.Done: //cинхронизация с закрытием канала, если получен os.Exit
			return
		case h.Pool.Channel <- task:
		}
	}
}

// StatusCheck provides task info if it exists. Field Archive is passed only if the task is complete.
func (h *TasksHandler) StatusCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}
	tid := chi.URLParam(r, "id")
	if tid == "" {
		http.Error(w, "Empty task ID", http.StatusBadRequest)
		return
	}
	h.Pool.RLock()
	task, exists := h.Pool.Mapa[tid]
	h.Pool.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("Task with ID '%s' doesn't exist", tid), http.StatusNotFound)
		return
	}
	task.RLock()
	defer task.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(task); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode task '%v' info", tid), http.StatusInternalServerError)
		return
	}
}

// ReturnArchive returns an archive according to task id and archive name
func (h *TasksHandler) ReturnArchive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}
	taskID := chi.URLParam(r, "task_id")
	archName := chi.URLParam(r, "file_name")
	if taskID == "" || archName == "" {
		http.Error(w, "Invalid link to archive", http.StatusBadRequest)
		return
	}
	filePath := filepath.Join(h.Pool.ArchDir, taskID, archName)

	http.ServeFile(w, r, filePath)
}
