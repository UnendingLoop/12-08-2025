package main

import (
	"09-09-2025/cmd/handler"
	"09-09-2025/cmd/model"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {

	r := chi.NewRouter()
	taskHandler := handler.TasksHandler{Pool: &model.TasksMap{Mapa: make(map[string]*model.Task), Channel: make(chan *model.Task)}}
	defer close(taskHandler.Pool.Channel)

	r.Post("/tasks", taskHandler.CreateNewTask)      //создание задачи, возвращает ID
	r.Get("/tasks/{id}", taskHandler.StatusCheck)    //получение статуса задачи(возможно со ссылкой на скачивание архива если готово)
	r.Post("/tasks/{id}", taskHandler.AddLinkToTask) //добавление ссылки на скачивание файла - 1 ссылка за раз

	//Starting server

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
	log.Printf("Server running on http://localhost:8080")
}
