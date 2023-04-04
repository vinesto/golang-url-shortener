package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/xid"
)

type URL struct {
	ID        string    `json:"id"`
	LongURL   string    `json:"long_url"`
	ShortURL  string    `json:"short_url"`
	CreatedAt time.Time `json:"created_at"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./urls.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id TEXT PRIMARY KEY,
			long_url TEXT NOT NULL,
			short_url TEXT NOT NULL,
			created_at TEXT NOT NULL,
			redirects INTEGER NOT NULL DEFAULT 0,
			last_access TEXT
		);
	`)
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/shorten", shortenURL).Methods("POST")
	router.HandleFunc("/{shortURL:[a-zA-Z0-9]{6}}", redirectToLongURL).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", router))
	fmt.Println("Server running on port 8080")
}

func shortenURL(w http.ResponseWriter, r *http.Request) {
	longURL := r.FormValue("long_url")
	if longURL == "" {
		http.Error(w, "Missing long_url parameter", http.StatusBadRequest)
		return
	}

	h := md5.Sum([]byte(longURL))
	id := hex.EncodeToString(h[:])
	shortURL := id[:6]

	url := URL{
		ID:        xid.New().String(),
		LongURL:   longURL,
		ShortURL:  shortURL,
		CreatedAt: time.Now(),
	}

	_, err := db.Exec("INSERT INTO urls (id, long_url, short_url, created_at) VALUES (?, ?, ?, ?)", url.ID, url.LongURL, url.ShortURL, url.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to save URL to database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"short_url": "http://localhost:8080/` + url.ShortURL + `"}`))
}

func redirectToLongURL(w http.ResponseWriter, r *http.Request) {
	shortURL := mux.Vars(r)["shortURL"]
	var longURL string
	err := db.QueryRow("SELECT long_url FROM urls WHERE short_url = ?", shortURL).Scan(&longURL)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, longURL, http.StatusSeeOther)
}
