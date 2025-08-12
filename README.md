# File Downloader Service

Сервис на Go для скачивания файлов по ссылкам, упаковки их в архив и раздачи готового результата через HTTP.  
Подходит как заготовка для бэкенд-приложений, где нужно асинхронно обрабатывать задачи и управлять процессом через API.

---

## Что умеет

- Принимать задачи на скачивание файлов.
- Добавлять ссылки в задачу по одной за 1 реквест.
- Запускать **3 горутины-воркера** для скачивания файлов (в будущем можно вынести в .env).
- Скачивание стартует **только после добавления третьей ссылки в задачу**.
- Файлы в задаче скачиваются **последовательно**.
- Если активных задач уже **3**, то новую задачу добавить невозможно.
- Создавать ZIP-архив по готовности загрузки файлов. Архив создастся даже если некоторые файлы не были скачаны.
- Отдавать архив клиенту по HTTP.
- Работать с конфигурацией через `.env` (порт, пути к папкам, допустимые расширения файлов).
- Корректно завершать работу (graceful shutdown) с очисткой временных файлов.

---

## Стек технологий

- **Go** — основной язык.
- **chi** — роутер для HTTP API.
- **sync.WaitGroup**, **chan** — управление конкурентностью.
- **atomic**, **RWmutex** — потокобезопасный доступ к данным.
- **os/signal**, **context** — graceful shutdown.
- **encoding/json** — сериализация/десериализация.
- **os.MkdirAll**, **archive/zip** — работа с файлами и архивами.

---

## Установка и запуск

1. Склонировать репозиторий:
   ```bash
   git clone https://github.com/UnendingLoop/09-09-2025.git
   cd 09-09-2025
   ```

2. Установить зависимости:
   ```bash
   go mod tidy
   ```

3. Создать файл `.env` в корне проекта (пример):
   ```env
   APP_PORT=:8080
   TMP_DIR=tmp
   ARCHIVE_DIR=archive
   VALID_EXTENTIONS='[".pdf",".jpg"]'
   ```

4. Запустить приложение:
   ```bash
   go run cmd/main.go
   ```
   Запуск из папки cmd не рекомендуется, так как в таком случае не считается env-file и приложение запустится с параметрами по умолчанию.

5. Сервер по умолчанию стартует по адресу:
   ```
   http://localhost:8080
   ```

---

## API

### Создать новую задачу
```
POST /tasks
```
Ответ:
```json
{
    "id": "4936970c-125a-403d-90ea-d552f93abf53",
    "files": [],
    "task_status": "pending"
}
```

---

### Добавить ссылку в задачу
```
POST /tasks/{id}
```
Тело запроса:
```json
{"file_URL":"https://example.com/file.pdf"}
```
Ответ: HTTP 204

Валидные на 12.08.2025 ссылки для тестирования(далее использованы в примерах):
https://www.postgresql.org/files/documentation/pdf/16/postgresql-16-A4.pdf
https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf
https://unec.edu.az/application/uploads/2014/12/pdf-sample.pdf

---

### Проверить статус задачи
```
GET /tasks/{id}
```
Ответ по неначатой задаче - добавлены только 2 ссылки из 3х:
```json
{
    "id": "4936970c-125a-403d-90ea-d552f93abf53",
    "files": [
        {
            "file_URL": "https://www.postgresql.org/files/documentation/pdf/16/postgresql-16-A4.pdf",
            "file_status": "pending"
        },
        {
            "file_URL": "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf",
            "file_status": "pending"
        }
    ],
    "task_status": "pending"
}
```
Ответ по завершенной задаче:
```json
{
    "id": "4936970c-125a-403d-90ea-d552f93abf53",
    "files": [
        {
            "file_URL": "https://www.postgresql.org/files/documentation/pdf/16/postgresql-16-A4.pdf",
            "file_status": "ready"
        },
        {
            "file_URL": "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf",
            "file_status": "ready"
        },
        {
            "file_URL": "https://unec.edu.az/application/uploads/2014/12/pdf-sample.pdf",
            "file_status": "ready"
        }
    ],
    "task_status": "ready",
    "archive_URI": "/archive/4936970c-125a-403d-90ea-d552f93abf53/archive.zip"
}
```
---

### Скачать готовый архив
```
GET /archive/{task_ID}/archive.zip
```

---

## Примеры работы через curl

1. Создаём задачу:
   ```bash
   curl -X POST http://localhost:8080/tasks
   ```

2. Добавляем ссылки:
   ```bash
   curl -X POST -H "Content-Type: application/json" -d '{"link":"https://www.postgresql.org/files/documentation/pdf/16/postgresql-16-A4.pdf"}' http://localhost:8080/tasks/<task_id>
   curl -X POST -H "Content-Type: application/json" -d '{"https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf"}' http://localhost:8080/tasks/<task_id>
   curl -X POST -H "Content-Type: application/json" -d '{"https://unec.edu.az/application/uploads/2014/12/pdf-sample.pdf"}' http://localhost:8080/tasks/<task_id>
   ```

3. Проверяем статус:
   ```bash
   curl http://localhost:8080/tasks/<task_id>
   ```

4. Скачиваем архив:
   ```bash
   curl -O http://localhost:8080/archive/<task_id>/archive.zip
   ```

---

## Структура проекта
```
cmd/
  handler/  — обработчики HTTP-запросов
  model/    — модели данных и структуры задач
  service/  — логика воркеров и скачивания
  main.go   — точка входа
config/     — чтение настроек из .env
```

---

## Graceful shutdown
При нажатии `Ctrl+C` сервер:
- Останавливает приём новых запросов.
- Ждёт завершения активных задач.
- Закрывает каналы.
- Удаляет временные файлы.

---

## Что можно улучшить
- Добавить сохранение и восстановление задач при перезапуске через файл.
- Покрыть код тестами.
- Сделать скачивание с проверкой контрольных сумм.
- Добавить авторизацию для API.
- Добавить Swagger-документацию.

---

Если кратко: этот сервис — про **надёжную обработку задач** и **чистое завершение работы** без потерь.
