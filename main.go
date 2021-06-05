package main

import (
    "fmt"
    "log"
		"net/http"
		"encoding/json"
		"github.com/gorilla/mux"
		"database/sql"
		_ "github.com/lib/pq"
		"io/ioutil"
		"strings"
)

const (  // Параметры для подключения к базе данных
  host     = "localhost"
  port     = 5432
  user     = "yourname"
  password = "yourpassword"
  dbname   = "postgres"
)
// подключение к бд
var psqlInfo = fmt.Sprintf("host=%s port=%d user=%s "+
"password=%s dbname=%s sslmode=disable",
host, port, user, password, dbname)

var db, err = sql.Open("postgres", psqlInfo)





type Forum struct {
	Title   string `json:"title"`
	User    string `json:"user"`
	Slug    string `json:"slug"`
	Posts   string `json:"posts"`
	Threads string `json:"threads"`
}

type Thread struct {
	Id      int32  `json:"id"` 
	Title   string `json:"title"`
	Author  string `json:"author"`
	Forum   string `json:"forum"`
	Message string `json:"message"`
	Votes   int32  `json:"votes"`
	Slug    string `json:"slug"`
	Created string `json:"created"`
}

func createForum(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)

	var forum Forum
	json.Unmarshal(reqBody, &forum)

	sqlString := fmt.Sprintf("INSERT INTO forums (username, slug, title) VALUES ('%s','%s','%s')", forum.User, forum.Slug, forum.Title)
	//sqlString := "INSERT INTO forums (username, slug, title) VALUES ('abc','abcde','abc')"
	_, err = db.Query(sqlString)
	if (err != nil) {
		fmt.Println(err)
		if strings.Contains(err.Error(), "insert or update") {
			http.Error(w, "Can't find user", 404)
			return
		}

		if strings.Contains(err.Error(), "duplicate key value") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
		}
	}

	sqlString2 := fmt.Sprintf("SELECT title, username as user, slug, posts, threads FROM forums WHERE slug='%s'", forum.Slug)
	rows, _:= db.Query(sqlString2)


	var got Forum
	for rows.Next() {
		forum := new(Forum)

		rows.Scan(&forum.Title, &forum.User, &forum.Slug, &forum.Posts, &forum.Threads)

		got = *forum
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(got)
}

func detailsForum(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]

	sql := fmt.Sprintf("SELECT title, username as user, slug, posts, threads FROM forums WHERE slug='%s'", slug)

	rows, _ := db.Query(sql)

	var got Forum
	for rows.Next() {
		forum := new(Forum)

		rows.Scan(&forum.Title, &forum.User, &forum.Slug, &forum.Posts, &forum.Threads)

		got = *forum
	}

	if (got.Slug == "") {
		http.Error(w, "Can't find forum", 404)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(got)
}

func forumThreadCreate(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)

	vars := mux.Vars(r)
	slug := vars["slug"]

	var thread Thread
	json.Unmarshal(reqBody, &thread)

	sql := fmt.Sprintf("SELECT id FROM forums WHERE slug='%s'", slug)

	
	rows, _ := db.Query(sql)

	var forumId int32
	for rows.Next() {
		rows.Scan(&forumId)
	}

	sql1 := fmt.Sprintf("SELECT nickname FROM users WHERE UPPER(nickname) = UPPER('%s')", thread.Author)

	
	rows1, _ := db.Query(sql1)

	var userNickname string
	for rows1.Next() {
		rows1.Scan(&userNickname)
	}

	if (forumId == 0 || userNickname == "") {
		http.Error(w, "Can't find", 404)
		return
	}

	sql2 := fmt.Sprintf("SELECT t.id, t.author, t.message, t.title, t.slug, t.created, f.slug as 'forum' FROM threads t INNER JOIN forums f ON(t.forum = f.id) WHERE t.slug = '%s'", slug)

	rows2, _ := db.Query(sql2)

	var threadOut Thread
	for rows.Next() {
		rows.Scan(&threadOut)
	}

	if (threadOut.Author != "") {
		//
	}
}

func handleRequests() { // РОУТЫ !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
  myRouter := mux.NewRouter().StrictSlash(true)

	myRouter.HandleFunc("/api/forum/create", createForum).Methods("POST")
	myRouter.HandleFunc("/api/forum/{slug}/details", detailsForum)
	myRouter.HandleFunc("/api/forum/{slug}/create", forumThreadCreate).Methods("POST")
  log.Fatal(http.ListenAndServe(":10000", myRouter))
}



func main() {
  err = db.Ping()
  if err != nil {
    panic(err)
	}
	
	handleRequests()
}
