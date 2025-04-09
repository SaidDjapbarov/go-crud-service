package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Ниже - небольшой пример CRUD-приложения.

// Book - модель данных, которую будем хранить в PostgreSQL.
type Book struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

// db - глобальная переменная для удобства.
var db *sql.DB

func main() {
	// Подключаемся к PostgreSQL, используя переменные окружения.
	//   POSTGRES_HOST=localhost
	//   POSTGRES_PORT=5432
	//   POSTGRES_USER=postgres
	//   POSTGRES_PASSWORD=password
	//   POSTGRES_DB=mydatabase
	pgHost := getEnv("POSTGRES_HOST", "localhost")
	pgPort := getEnv("POSTGRES_PORT", "5432")
	pgUser := getEnv("POSTGRES_USER", "postgres")
	pgPass := getEnv("POSTGRES_PASSWORD", "password")
	pgDB := getEnv("POSTGRES_DB", "postgres")

	// Формируем строку подключения.
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		pgHost, pgPort, pgUser, pgPass, pgDB)

	// Открываем соединение и проверяем на ошибки.
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка соединения с PostgreSQL: %v", err)
	}

	// Проверяем, что бд отвечает.
	if err = db.Ping(); err != nil {
		log.Fatalf("PostgreSQL не отвечает: %v", err)
	}

	// Создаем таблицу (если она не существует) для хранения книг.
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS books (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		author VARCHAR(255) NOT NULL,
		year INT NOT NULL
	);
	`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Не удалось создать таблицу: %v", err)
	}

	// Настраиваем роуты.
	http.HandleFunc("/books", handleBooks)     // для POST и GET (список)
	http.HandleFunc("/books/", handleBookByID) // для GET, PUT, DELETE (конкретная книга)

	// Запускаем HTTP-сервер на порту 8080.
	log.Println("Сервер запущен на http://localhost:8080 ...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// handleBooks — обрабатывает POST (создание книги) и GET (получение всех книг).
func handleBooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		createBook(w, r)
	case http.MethodGet:
		getAllBooks(w, r)
	default:
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}

// handleBookByID — обрабатывает GET, PUT и DELETE для /books/{id}.
func handleBookByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getBookByID(w, r)
	case http.MethodPut:
		updateBookByID(w, r)
	case http.MethodDelete:
		deleteBookByID(w, r)
	default:
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
	}
}

// createBook — пример "Create" в CRUD. Принимает JSON с Title, Author, Year,
// создает новую запись в таблице books и возвращает её ID.
func createBook(w http.ResponseWriter, r *http.Request) {
	var b Book

	// Парсим JSON из тела запроса
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "Невалидный JSON", http.StatusBadRequest)
		return
	}

	// Для безопасности: если чего-то не хватает, говорим пользователю об этом
	if b.Title == "" || b.Author == "" {
		http.Error(w, "Не хватает полей Title или Author", http.StatusBadRequest)
		return
	}

	// Создаем контекст с таймаутом.
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	query := `INSERT INTO books (title, author, year) VALUES ($1, $2, $3) RETURNING id;`
	err := db.QueryRowContext(ctx, query, b.Title, b.Author, b.Year).Scan(&b.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка добавления книги: %v", err), http.StatusInternalServerError)
		return
	}

	// Возвращаем в JSON созданную книгу (уже с ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
}

// getAllBooks — пример "Read" (список) из CRUD. Возвращаем все книги.
func getAllBooks(w http.ResponseWriter, r *http.Request) {
	// Создаем контекст с таймаутом.
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `SELECT id, title, author, year FROM books`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка выборки книг: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Year); err != nil {
			http.Error(w, fmt.Sprintf("Ошибка чтения строки: %v", err), http.StatusInternalServerError)
			return
		}
		books = append(books, b)
	}

	// Возвращаем JSON со списком
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(books)
}

// getBookByID — пример "Read" (конкретная запись) из CRUD.
func getBookByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromURL(r)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	// Короткий контекст с timeout
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var b Book
	query := `SELECT id, title, author, year FROM books WHERE id = $1`
	err = db.QueryRowContext(ctx, query, id).Scan(&b.ID, &b.Title, &b.Author, &b.Year)
	if err == sql.ErrNoRows {
		http.Error(w, "Книга не найдена", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка выборки книги: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
}

// updateBookByID — пример "Update" из CRUD. Обновляем поля книги по ID.
func updateBookByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromURL(r)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	var b Book
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "Невалидный JSON", http.StatusBadRequest)
		return
	}

	// Для упрощения — не проверяем пустые поля.
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	query := `UPDATE books SET title=$1, author=$2, year=$3 WHERE id=$4`
	res, err := db.ExecContext(ctx, query, b.Title, b.Author, b.Year, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка обновления: %v", err), http.StatusInternalServerError)
		return
	}

	// Проверяем, что обновилась хотя бы 1 строка
	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.Error(w, "Книга с таким ID не найдена", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Книга %d успешно обновлена\n", id)
}

// deleteBookByID — пример "Delete" из CRUD. Удаляем запись по ID.
func deleteBookByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromURL(r)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	query := `DELETE FROM books WHERE id=$1`
	res, err := db.ExecContext(ctx, query, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка удаления: %v", err), http.StatusInternalServerError)
		return
	}

	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.Error(w, "Книга с таким ID не найдена", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Книга %d успешно удалена\n", id)
}

// parseIDFromURL — вспомогательная функция для извлечения ID (int64)
func parseIDFromURL(r *http.Request) (int64, error) {
	// /books/123 => parts[0] = "", parts[1] = "books", parts[2] = "123"
	parts := splitPath(r.URL.Path)
	if len(parts) < 3 {
		return 0, fmt.Errorf("неверный путь: %s", r.URL.Path)
	}
	var id int64
	_, err := fmt.Sscan(parts[2], &id)
	return id, err
}

// splitPath — простейшая функция для разбиения URL Path.
// /books/123 => ["", "books", "123"].
func splitPath(path string) []string {
	return []string{
		// Разделим вручную, чтобы не тянуть лишние библиотеки.
		// Или можно strings.Split(path, "/").
		// Но учтём, что при Split("/books/123", "/") в начале будет "".
		"",
		"books",
		path[len("/books/"):],
	}
}

// getEnv — функция для чтения переменных окружения с умолчаниями.
func getEnv(key, defVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defVal
	}
	return val
}
