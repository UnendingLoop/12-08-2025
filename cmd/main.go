package main

import (
	"09-09-2025/cmd/handler"
	"09-09-2025/cmd/model"
	"09-09-2025/cmd/service"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	//создание папки archive, если её нет
	if err := os.MkdirAll("archive", 0755); err != nil {
		log.Fatalf("Failed to create archive directory: %v", err)
	}

	wg := sync.WaitGroup{}

	r := chi.NewRouter()
	taskHandler := handler.TasksHandler{Pool: &model.TasksMap{Mapa: make(map[string]*model.Task), Channel: make(chan *model.Task, 9), Done: make(chan struct{})}}

	r.Post("/tasks", taskHandler.CreateNewTask)                 //создание задачи, возвращает ID
	r.Get("/tasks/{id}", taskHandler.StatusCheck)               //получение статуса задачи(возможно со ссылкой на скачивание архива если готово)
	r.Post("/tasks/{id}", taskHandler.AddLinkToTask)            //добавление ссылки на скачивание файла - 1 ссылка за раз
	r.Get("/archive/{archive_name}", taskHandler.ReturnArchive) //скачивание архива

	//Starting server
	srv := http.Server{Addr: ":8080", Handler: r}
	go func() {
		log.Printf("Launching server on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Server stopped: %v", err)
		}
	}()

	log.Printf("Server launched successfully!")

	//Starting shutdown signal listener
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	defer close(sig)

	wg.Add(1)
	go func() {
		<-sig
		log.Println("Interrupt received. Starting shutdown sequence...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		log.Println("HTTP server stopped")
		close(taskHandler.Pool.Done)
		close(taskHandler.Pool.Channel)
		wg.Done()
	}()

	//Starting TaskManager - 3 workers
	wg.Add(3)
	for i := range 3 {
		log.Println("Started goroutine", i)
		go service.TaskManager(&wg, taskHandler.Pool.Channel, &taskHandler.Pool.ActiveTasksCount)
	}
	wg.Wait()
}
