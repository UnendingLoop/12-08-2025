package service

import (
	"09-09-2025/cmd/model"
	"archive/zip"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
)

// TaskManager обрабатывает задачи из канала
func TaskManager(wg *sync.WaitGroup, input chan *model.Task, activeTasksCounter *atomic.Int32) {
	defer wg.Done()
	for task := range input {
		if task == nil {
			continue
		}
		activeTasksCounter.Add(1)

		task.Lock()
		task.Status = model.StatusInProgress
		internalTask := *task
		//при копировании структуры копии поля Files не будет так как это массив
		//поэтому надо явно скопировать его
		tempFiles := []*model.FileInfo{}
		for _, v := range task.Files {
			tempFile := *v
			tempFiles = append(tempFiles, &tempFile)
		}
		task.Unlock()
		internalTask.Files = tempFiles

		// Создание временной папки для файлов задачи
		tempDir := filepath.Join(internalTask.TmpDir, internalTask.TID)
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			log.Println("failed to create temp dir:", err)

			task.Lock()
			task.Status = model.StatusError
			task.Unlock()

			activeTasksCounter.Add(-1)
			continue
		}

		// Скачивание
		for i, file := range internalTask.Files {
			downloader(file, tempDir, i)
		}
		task.Lock()
		task.Files = internalTask.Files //для обновления статусов файлов
		task.Unlock()

		// Архивирование
		//создание индивидуальной папки для архива
		archDir := filepath.Join(internalTask.ArchDir, internalTask.TID)
		if err := os.MkdirAll(archDir, 0755); err != nil {
			log.Fatalf("Failed to create unique ARCHIVE directory: %v", err)
			internalTask.Status = model.StatusError
			activeTasksCounter.Add(-1)
			continue
		}
		archName := "archive.zip"
		archPath := filepath.Join(archDir, archName)

		if archFiles, err := archiver(internalTask.Files, internalTask.TID, archPath, tempDir); len(err) != 0 {
			log.Printf("Problems while creating archive for task '%s':%v", internalTask.TID, err)
			internalTask.Files = archFiles
			internalTask.Status = model.StatusError
			activeTasksCounter.Add(-1)
			continue
		}

		// Обновляем ссылку на архив — для отдачи клиенту
		archLink := "/" + archPath
		internalTask.Archive = &archLink
		internalTask.Status = model.StatusReady

		activeTasksCounter.Add(-1)

		task.Lock()
		task.Files = internalTask.Files
		task.Status = internalTask.Status
		task.Archive = internalTask.Archive
		task.Unlock()
	}
}

// downloader скачивает файл в destDir с уникальным префиксом
func downloader(file *model.FileInfo, destDir string, prefix int) {
	resp, err := http.Get(file.URL)
	if err != nil {
		log.Println(err)
		file.Status = model.StatusError
		file.Error = &model.ErrDownloadFailed
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		file.Status = model.StatusError
		file.Error = &model.ErrDownloadFailed
		log.Printf("File %v failed to download:\n%v", file.Name, resp.StatusCode)
		return
	}

	// берем имя файла из URL или хедера
	filename := path.Base(file.URL)
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if fname, ok := params["filename"]; ok {
				filename = fname
			}
		}
	}
	file.Name = strconv.Itoa(prefix) + "_" + filename

	destPath := filepath.Join(destDir, file.Name)
	out, err := os.Create(destPath)
	if err != nil {
		file.Status = model.StatusError
		log.Printf("Failed to create new file(for file '%s') in local storage:\n%v", file.Name, err)
		return
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		file.Status = model.StatusError
		log.Printf("Failed to save r.Body of file '%s' to local storage:\n%v", file.Name, resp.StatusCode)
		return
	}
	file.Status = model.StatusReady

}

// archiver архивирует успешно скачанные файлы и добавляет их в архив
func archiver(files []*model.FileInfo, tid, destZip, baseDir string) ([]*model.FileInfo, []error) {
	errors := make([]error, 0, 7)
	zipFile, err := os.Create(destZip)
	if err != nil {
		errors = append(errors, err)
		return files, errors
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, file := range files {
		if file.Status != model.StatusReady {
			continue
		}

		fullPath := filepath.Join(baseDir, file.Name)
		f, err := os.Open(fullPath)
		if err != nil {
			log.Printf("Warning task ID '%v'! Failed to open file '%s': %v", tid, fullPath, err)
			errors = append(errors, err)
			continue
		}

		info, err := f.Stat()
		if err != nil {
			log.Printf("Warning task ID '%v'! Failed to read file '%s': %v", tid, fullPath, err)
			f.Close()
			errors = append(errors, err)
			continue
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			log.Printf("Warning task ID '%v'! Failed to read file '%s': %v", tid, fullPath, err)
			f.Close()
			errors = append(errors, err)
			continue
		}
		header.Name = file.Name

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			log.Printf("Warning task ID '%v'! Failed to create archive '%s': %v", tid, fullPath, err)
			f.Close()
			errors = append(errors, err)
			continue
		}

		if _, err = io.Copy(writer, f); err != nil {
			log.Printf("Warning task ID '%v'! Failed to write compressed file '%s' to archive: %v", tid, fullPath, err)
			f.Close()
			errors = append(errors, err)
			continue
		}
		f.Close()
	}
	return files, errors
}
