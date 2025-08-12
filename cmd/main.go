package main

import (
	"09-09-2025/cmd/handler"
	"09-09-2025/cmd/model"
	"09-09-2025/cmd/service"
	"09-09-2025/config"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	//загрузка параметров из env. и создание временной папки и папки для хранения архивов
	config := config.GetConfig()

	wg := sync.WaitGroup{}

	r := chi.NewRouter()
	taskHandler := handler.TasksHandler{Pool: &model.TasksMap{Mapa: make(map[string]*model.Task), Channel: make(chan *model.Task, 9), Done: make(chan struct{}), ValidExt: config.ValidExt, ArchDir: config.ArchDir, TmpDir: config.TmpDir}}

	r.Post("/tasks", taskHandler.CreateNewTask)                        //создание задачи, возвращает ID
	r.Get("/tasks/{id}", taskHandler.StatusCheck)                      //получение статуса задачи(возможно со ссылкой на скачивание архива если готово)
	r.Post("/tasks/{id}", taskHandler.AddLinkToTask)                   //добавление ссылки на скачивание файла - 1 ссылка за раз
	r.Get("/archive/{task_id}/{file_name}", taskHandler.ReturnArchive) //скачивание архива

	//Starting server
	srv := http.Server{
		Addr:         (":" + strconv.Itoa(config.AppPort)),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second}

	go func() {
		log.Printf("Launching server on http://localhost:%d", config.AppPort)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server stopped: %v", err)
		}
		log.Println("Server gracefully stopping...")
	}()

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
		log.Println("HTTP server stopped, waiting for goroutines...")
		close(taskHandler.Pool.Done)
		close(taskHandler.Pool.Channel)
		wg.Done()
		log.Println("Goroutines finished.\nRemoving TEMP-directory...")
		err := os.RemoveAll(config.TmpDir)
		if err != nil {
			log.Printf("Failed to remove TEMP-directory: %v\nExiting application.", err)
			return
		}
		log.Printf("Removing DONE\nExiting application.")
	}()

	//Starting TaskManager - 3 workers
	wg.Add(3)
	for i := range 3 {
		log.Println("Started goroutine", i)
		go service.TaskManager(&wg, taskHandler.Pool.Channel, &taskHandler.Pool.ActiveTasksCount)
	}
	wg.Wait()
}
