package handler

import (
	"09-09-2025/cmd/model"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TasksHandler struct {
	Pool *model.TasksMap
}

// CreateNewTask - creates a task in model.TasksMap.Mapa if there are less than 3 tasks in progress
func (h *TasksHandler) CreateNewTask(w http.ResponseWriter, r *http.Request) {
	if h.Pool.ActiveTasksCount.Load() >= 3 {
		http.Error(w, fmt.Sprintf("Failed to create new task: %v", model.ErrBusy), http.StatusServiceUnavailable)
		return
	}
	newID := uuid.New().String()
	newTask := &model.Task{
		TID:        newID,
		FilesCount: 0,
		Files:      []model.FileInfo{},
		Status:     model.StatusPending}

	h.Pool.Lock()
	h.Pool.Mapa[newID] = newTask
	h.Pool.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(newTask); err != nil {
		http.Error(w, "Failed to encode task info", http.StatusInternalServerError)
		return
	}
}

// AddLinkToTask checks if such task exists by ID from path, creates model.FileInfo and adds to task. If there are 3 files in task - sends it to chan
func (h *TasksHandler) AddLinkToTask(w http.ResponseWriter, r *http.Request) {
	tid := chi.URLParam(r, "id")
	if tid == "" {
		http.Error(w, "Empty task ID", http.StatusBadRequest)
	}

	h.Pool.RLock()
	defer h.Pool.Unlock()

	task, exists := h.Pool.Mapa[tid]
	if !exists {
		http.Error(w, fmt.Sprintf("Task with ID '%s' doesn't exist", tid), http.StatusNotFound)
		return
	}
	if task.FilesCount >= 3 {
		http.Error(w, fmt.Sprintf("Failed to add link to task '%s': %v", tid, model.ErrTaskIsFull), http.StatusConflict)
		return
	}
	newFile := model.FileInfo{}
	if err := json.NewDecoder(r.Body).Decode(&newFile); err != nil { //должен считаться только URL
		http.Error(w, fmt.Sprintf("Failed to decode task '%s' info", tid), http.StatusInternalServerError)
		return
	}

	newFile.URL = strings.TrimSpace(newFile.URL)

	//валидация ссылки - проверка на уникальность внутри одной таски
	for _, v := range task.Files {
		if v.URL == newFile.URL {
			http.Error(w, fmt.Sprintf("Link '%s' already in task '%s'", newFile.URL, tid), http.StatusBadRequest)
			return
		}
	}

	newFile.Status = model.StatusPending
	task.Files = append(task.Files, newFile)
	task.FilesCount++

	if task.FilesCount >= 3 {
		h.Pool.Channel <- task
	}

	//создать экземпляр fileinfo и положить его в таску по uuid
	//как раз здесь проверка на кол-во файлов: если 3 - передаем в канал для обработки воркерами

}

func (h *TasksHandler) StatusCheck(w http.ResponseWriter, r *http.Request) {
	var newTask model.Task
	if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
	}
}
