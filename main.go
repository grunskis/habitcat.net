package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	_ "github.com/lib/pq"
)

type viewHandler func(w http.ResponseWriter, r *http.Request)

var hashKey = []byte{98, 231, 101, 158, 43, 6, 214, 248, 106, 188, 241, 109, 239, 5, 242, 221, 159, 154, 157, 87, 4, 184, 232, 107, 126, 71, 84, 67, 61, 189, 160, 65}
var blockKey = []byte{84, 185, 64, 70, 61, 137, 3, 132, 95, 215, 51, 249, 142, 19, 209, 146}

var db *sql.DB

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	var err error
	db, err = createDBConnection("activities")
	if err != nil {
		log.Fatalln("opening db connection failed:", err)
	}
	defer db.Close()

	http.HandleFunc("/", indexHandler)

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/signup", signupHandler)

	http.HandleFunc("/goals", authHandler(goalHandler))
	http.HandleFunc("/goals/", authHandler(goalUpdateHandler))
	http.HandleFunc("/goals/new", authHandler(goalNewHandler))
	http.HandleFunc("/goals/create", authHandler(goalCreateHandler))

	http.HandleFunc("/habits", authHandler(habitHandler))
	http.HandleFunc("/habits/", authHandler(habitUpdateHandler))
	http.HandleFunc("/habits/new", authHandler(habitNewHandler))
	http.HandleFunc("/habits/create", authHandler(habitCreateHandler))

	staticFileServer := http.StripPrefix("/static/", http.FileServer(http.Dir("./static/")))
	http.Handle("/static/", staticFileServer)
	http.Handle("/favicon.ico", staticFileServer)

	log.Println("Server listening on http://0.0.0.0:" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func createDBConnection(dbname string) (*sql.DB, error) {
	var dsn string
	_, found := os.LookupEnv("DATABASE_POSTGRESQL_USERNAME")
	if found {
		// for running test on semaphore ci
		dsn = fmt.Sprintf("dbname=%s user=runner password=semaphoredb sslmode=disable", dbname)
	} else {
		dsn, found = os.LookupEnv("DATABASE_URL")
		if !found {
			dsn = fmt.Sprintf("dbname=%s user=postgres sslmode=disable", dbname)
		}
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func renderTemplate(w http.ResponseWriter, templateFilename string, data interface{}) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles(templateFilename)
	if err != nil {
		log.Fatal(err)
	}

	err = t.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

func renderTemplateWithErrorMessage(w http.ResponseWriter, templatePath string, msg string) {
	data := struct {
		ErrorMessage string
	}{
		msg,
	}
	renderTemplate(w, templatePath, data)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "templates/index.html", nil)
}

func authHandler(next viewHandler) viewHandler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("habitcat")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		var s = securecookie.New(hashKey, blockKey)
		value := make(map[string]string)
		if err := s.Decode("habitcat", cookie.Value, &value); err != nil {
			log.Println("Failed to decode cookie")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// Set current users account ID in the request context
		context.Set(r, "accountId", value["accountId"])
		defer context.Clear(r) // clear request context after request is handled

		next(w, r)
	}

	return fn
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderTemplate(w, "templates/login.html", nil)
	} else if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		email, password := r.FormValue("email"), r.FormValue("password")
		account, err := GetAccount(email)
		if err != nil {
			log.Println(err)
			msg := "Email/password combination incorrect..."
			renderTemplateWithErrorMessage(w, "templates/login.html", msg)
			return
		}
		if account.ValidatePassword([]byte(password)) {
			var s = securecookie.New(hashKey, blockKey)

			value := map[string]string{
				"accountId": account.Id,
			}

			if encoded, err := s.Encode("habitcat", value); err == nil {
				cookie := &http.Cookie{Name: "habitcat", Value: encoded, Path: "/"}
				http.SetCookie(w, cookie)
				http.Redirect(w, r, "/habits", http.StatusFound)
				return
			} else {
				log.Println(err)
				return
			}
		} else {
			log.Println("Incorrect credentials for user", email)
			msg := "Email/password combination incorrect..."
			renderTemplateWithErrorMessage(w, "templates/login.html", msg)
			return
		}
	} else {
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "habitcat", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderTemplate(w, "templates/signup.html", nil)
	} else if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		email, password := r.FormValue("email"), r.FormValue("password")
		if !emailInvited(email) {
			log.Println("Somebody tried to sign up with disallowed email", email)
			data := struct {
				ErrorMessage string
			}{
				"Bummer! Your email is not on the invitation list...",
			}
			renderTemplate(w, "templates/signup.html", data)
			return
		}
		account, err := CreateAccount(email, password)
		if err != nil {
			log.Println("CreateAccount() failed", err)
			msg := "Account with this email already exists..."
			renderTemplateWithErrorMessage(w, "templates/signup.html", msg)
			return
		}

		var s = securecookie.New(hashKey, blockKey)

		value := map[string]string{
			"accountId": account.Id,
		}

		if encoded, err := s.Encode("habitcat", value); err == nil {
			cookie := &http.Cookie{Name: "habitcat", Value: encoded, Path: "/"}
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/habits", http.StatusFound)
		} else {
			log.Println(err)
		}
	} else {
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func emailInvited(email string) bool {
	invites := [2]string{"martins@grunskis.com", "antonellatezza@gmail.com"}
	for _, e := range invites {
		if email == e {
			return true
		}
	}
	return false
}
