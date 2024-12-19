package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type List struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

type Film struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Director    string   `json:"director"`
	Year        int      `json:"year"`
	Description string   `json:"description"`
	Reviews     []Review `json:"reviews" db:"-"`
}

type Review struct {
	ID     int    `json:"id"`
	UserID int    `json:"user_id"`
	FilmID int    `json:"film_id"`
	Review string `json:"review"`
	User   string `json:"user" db:"-"`
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	autoMigrate()

	http.HandleFunc("/register", register)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/film", film)
	http.HandleFunc("/review", review)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func autoMigrate() {
	createUsersTableSQL := `CREATE TABLE IF NOT EXISTS users (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"username" TEXT,
		"password" TEXT		
	  );`

	createFilmsTableSQL := `CREATE TABLE IF NOT EXISTS films (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"title" TEXT,
		"director" TEXT,
		"year" integer,
		"description" TEXT
	  );`

	createReviewsTableSQL := `CREATE TABLE IF NOT EXISTS reviews (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"film_id" integer,
		"review" TEXT,
		"user_id" integer,
		FOREIGN KEY(film_id) REFERENCES films(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	  );`

	statement, err := db.Prepare(createUsersTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()

	statement, err = db.Prepare(createFilmsTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()

	statement, err = db.Prepare(createReviewsTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec()
}

func register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if the username exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", user.Username).Scan(&count)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if count > 0 {
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", user.Username, string(hashedPassword))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var user User

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var storedPassword string
	err := db.QueryRow("SELECT password FROM users WHERE username = ?", user.Username).Scan(&storedPassword)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(user.Password)); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Create a session cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   user.Username,
		Expires: time.Now().Add(24 * time.Hour),
	})

	w.WriteHeader(http.StatusOK)
}

func logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Invalidate the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Expires: time.Now().Add(-1 * time.Hour),
	})

	w.WriteHeader(http.StatusOK)
}

func film(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Check if the user is logged in
	_, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if the "id" parameter is present in the URL
	idParam := r.URL.Query().Get("id")
	if idParam != "" {
		// Fetch the film with the specified ID
		var film Film
		err := db.QueryRow("SELECT id, title, director, year, description FROM films WHERE id = ?", idParam).Scan(&film.ID, &film.Title, &film.Director, &film.Year, &film.Description)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Film not found", http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Fetch the reviews for the film
		rows, err := db.Query("SELECT id, review, film_id, user_id FROM reviews WHERE film_id = ?", film.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var reviews []Review
		for rows.Next() {
			var review Review
			if err := rows.Scan(&review.ID, &review.Review, &review.FilmID, &review.UserID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var user User
			err := db.QueryRow("SELECT username FROM users WHERE id = ?", review.UserID).Scan(&user.Username)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			review.User = user.Username
			reviews = append(reviews, review)
		}

		film.Reviews = reviews
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(film)
		return
	}

	// Fetch the list of films
	rows, err := db.Query("SELECT id, title FROM films")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var films []List
	for rows.Next() {
		var film List
		if err := rows.Scan(&film.ID, &film.Title); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		films = append(films, film)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(films)
}

func review(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Check if the user is logged in
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var user User
		err = db.QueryRow("SELECT id FROM users WHERE username = ?", cookie.Value).Scan(&user.ID)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var review Review
		if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if review.FilmID == 0 || review.Review == "" {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		var film int
		db.QueryRow("SELECT id FROM films WHERE id = ?", review.FilmID).Scan(&film)
		if film == 0 {
			http.Error(w, "Film not found", http.StatusBadRequest)
			return
		}

		_, err = db.Exec("INSERT INTO reviews (film_id, review, user_id) VALUES (?, ?, ?)", review.FilmID, review.Review, user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	case http.MethodPatch:
		// Check if the user is logged in
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var user User
		err = db.QueryRow("SELECT id FROM users WHERE username = ?", cookie.Value).Scan(&user.ID)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var review Review
		if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		res, err := db.Exec("UPDATE reviews SET review = ? WHERE id = ? AND user_id = ?", review.Review, review.ID, user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		affected, err := res.RowsAffected()
		if affected == 0 {
			http.Error(w, "Unauthorized or Review not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	case http.MethodDelete:
		// Check if the user is logged in
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var user User
		err = db.QueryRow("SELECT id FROM users WHERE username = ?", cookie.Value).Scan(&user.ID)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var review Review
		if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		res, err := db.Exec("DELETE FROM reviews WHERE id = ? AND user_id = ?", review.ID, user.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		affected, err := res.RowsAffected()
		if affected == 0 {
			http.Error(w, "Unauthorized or Review not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
