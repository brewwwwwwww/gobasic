package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Book struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

const bookPath = "books"

var Db *sql.DB

const apibasePath = "/api"

func getBook(bookid int) (*Book, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := Db.QueryRowContext(ctx, `SELECT * FROM books WHERE id = ?`, bookid)

	book := &Book{}
	err := row.Scan(
		&book.ID,
		&book.Title,
		&book.Author,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		log.Println(err)
		return nil, err
	}
	return book, nil
}

func removeBook(bookID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := Db.ExecContext(ctx, `DELETE FROM books WHERE id = ?`, bookID)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func getBookList() ([]Book, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, err := Db.QueryContext(ctx, `SELECT * FROM books`)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer results.Close()
	books := make([]Book, 0)
	for results.Next() {
		var book Book
		results.Scan(&book.ID,
			&book.Title,
			&book.Author)

		books = append(books, book)
	}
	return books, nil
}

func insertBook(book Book) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := Db.ExecContext(ctx, `INSERT INTO books 
	(id,
	title,
	author
	)VALUES (?, ?, ?)`,
		book.ID,
		book.Title,
		book.Author)
	if err != nil {
		log.Println(err.Error())
		return 0, err
	}
	insertID, err := result.LastInsertId()
	if err != nil {
		log.Println(err.Error())
		return 0, err
	}
	return int(insertID), nil
}

func handlerBooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		BookList, err := getBookList()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json, err := json.Marshal(BookList)
		if err != nil {
			log.Fatal(err)
		}
		_, err = w.Write(json)
		if err != nil {
			log.Fatal(err)
		}
	case http.MethodPost:
		var book Book
		err := json.NewDecoder(r.Body).Decode(&book)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		BookID, err := insertBook(book)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(`{"bookid": %d}`, BookID)))
	case http.MethodOptions:
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handlerBook(w http.ResponseWriter, r *http.Request) {
	urlPathSegments := strings.Split(r.URL.Path, fmt.Sprintf("%s/", bookPath))
	if len(urlPathSegments[1:]) > 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	bookID, err := strconv.Atoi(urlPathSegments[len(urlPathSegments)-1])
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		book, err := getBook(bookID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if book == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json, err := json.Marshal(book)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = w.Write(json)
		if err != nil {
			log.Fatal(err)
		}
	case http.MethodDelete:
		err := removeBook(bookID)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/่json")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Origin, X-Requested-With")
		handler.ServeHTTP(w, r)
	})
}

func SetupRoutes(apiBasePath string) {
	bookHandler := http.HandlerFunc(handlerBook)
	http.Handle(fmt.Sprintf("%s/%s/", apiBasePath, bookPath), corsMiddleware(bookHandler))
	booksHandler := http.HandlerFunc(handlerBooks)
	http.Handle(fmt.Sprintf("%s/%s", apiBasePath, bookPath), corsMiddleware(booksHandler))
}

func SetupDB() {
	var err error
	Db, err = sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/bookdb")
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("success")
	}
	fmt.Println(Db)
	Db.SetConnMaxLifetime(time.Minute * 3)
	Db.SetMaxOpenConns(10)
	Db.SetMaxIdleConns(10)
}

func main() {
	SetupDB()
	SetupRoutes(apibasePath)
	log.Fatal(http.ListenAndServe(":5000", nil))
}
