# Go CRUD Service

Небольшой демо-проект на Go, реализующий CRUD-приложение для хранения книг.

## Стек

- **Go** (версия 1.23.5)
- **PostgreSQL** (через `github.com/lib/pq`)
- Стандартный `net/http` для REST
- **Контекст** (context) для ограничения времени запросов к БД

## Запуск

1. Установите переменные окружения (например, `POSTGRES_HOST`, `POSTGRES_PORT`, и т.д.).
2. Выполните:
    ```bash
    go mod tidy
    go run main.go
    ```
3. Приложение доступно на `http://localhost:8080`

## Примеры запросов

- **POST /books** (создать книгу):
  ```bash
  curl -X POST -H "Content-Type: application/json" \
       -d '{"title":"Golang Book","author":"John Doe","year":2023}' \
       http://localhost:8080/books
  ```

- **PUT /books** (обновить книгу):
  ```bash
  curl -X PUT -H "Content-Type: application/json" \
  -d '{"title":"Updated Title","author":"Another Author","year":2025}' \
  http://localhost:8080/books/1
  ```

- **GET /books/1** (полуить книгу по ID):
  ```bash
  curl http://localhost:8080/books/1
  ```

- **DELETE /books/1** (удалить книгу):
  ```bash
  curl -X DELETE http://localhost:8080/books/1
  ```
  
