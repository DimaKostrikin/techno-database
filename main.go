package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const ( // Параметры для подключения к базе данных
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
	ID      int32           `json:"id"`
	Title   string          `json:"title"`
	Author  string          `json:"author"`
	Forum   string          `json:"forum"`
	Message string          `json:"message"`
	Votes   int32           `json:"votes"`
	Slug    string          `json:"slug"`
	Created strfmt.DateTime `json:"created"`
}

type User struct {
	Nickname string `json:"nickname"`
	Fullname string `json:"fullname"`
	About    string `json:"about"`
	Email    string `json:"email"`
}

type Post struct {
	ID       int64           `json:"id"`
	Parent   int64           `json:"parent"`
	Author   string          `json:"author"`
	Message  string          `json:"message"`
	IsEdited bool            `json:"isEdited"`
	Forum    string          `json:"forum"`
	Thread   int32           `json:"thread"`
	Created  strfmt.DateTime `json:"Created"`
}

type PostFull struct {
	Post   Post   `json:"post"`
	Author User   `json:"author"`
	Thread Thread `json:"thread"`
	Forum  Forum  `json:"forum"`
}

type PostUpdate struct {
	Message string `json:"message"`
}

type Status struct {
	User   int32 `json:"user"`
	Forum  int32 `json:"forum"`
	Thread int32 `json:"thread"`
	Post   int64 `json:"post"`
}

/*
1111
  11
  11
  11
111111
*/

func createForum(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)

	var forum Forum
	json.Unmarshal(reqBody, &forum)

	sqlString := fmt.Sprintf("INSERT INTO forums (username, slug, title) VALUES ('%s','%s','%s')", forum.User, forum.Slug, forum.Title)
	//sqlString := "INSERT INTO forums (username, slug, title) VALUES ('abc','abcde','abc')"
	_, err = db.Query(sqlString)
	if err != nil {
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
	rows, _ := db.Query(sqlString2)

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

/*
 2222
22  22
   22
  22
222222
*/

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

	if got.Slug == "" {
		http.Error(w, "Can't find forum", 404)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(got)
}

/*
 3333
33  33
   333
33  33
 3333
*/

func forumThreadCreate(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)

	vars := mux.Vars(r)
	slug := vars["slug"]

	var thread Thread
	json.Unmarshal(reqBody, &thread)

	sql := fmt.Sprintf("INSERT INTO threads (author, forum, message, title, slug, created) VALUES ('%s', '%s', '%s', '%s', NULLIF('%s', ''), '%s') RETURNING id", thread.Author, slug, thread.Message, thread.Title, thread.Slug, thread.Created)

	insertId := 0
	err := db.QueryRow(sql).Scan(&insertId)

	if err != nil {
		fmt.Println(err)
		if strings.Contains(err.Error(), "violates foreign key") {
			http.Error(w, "Can't find forum", 404)
			return
		}

		if strings.Contains(err.Error(), "duplicate key value") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
		}
	}

	sql2 := fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE id='%d'", insertId)
	err = db.QueryRow(sql2).Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)

	sql3 := fmt.Sprintf("INSERT INTO usersForums (username, forum) VALUES ('%s','%s')", thread.Author, slug)
	db.Query(sql3)

	sql4 := fmt.Sprintf("UPDATE forums SET threads = threads + 1 WHERE slug = '%s'", slug)
	db.Query(sql4)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(thread)
}

/*
44  44
44  44
444444
    44
		44
*/

func forumThreadUsers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]

	sql := fmt.Sprintf("SELECT nickname, fullname, email, about FROM usersForums JOIN users ON (usersForums.username = users.nickname) WHERE usersForums.forum = '%s'", slug)

	rows, err := db.Query(sql)

	if err != nil {
		fmt.Print(err)
	}

	var got []User
	for rows.Next() {
		user := new(User)

		rows.Scan(&user.Nickname, &user.Fullname, &user.Email, &user.About)

		got = append(got, *user)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(got)
}

/*
555555
55
55555
    55
55555
*/

func getForumThreads(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug"]

	sql := fmt.Sprintf("SELECT id, title, author, forum, message, votes, slug, created FROM threads WHERE forum = '%s'", slug)

	rows, err := db.Query(sql)

	if err != nil {
		fmt.Print(err)
	}

	var got []Thread
	for rows.Next() {
		thread := new(Thread)

		rows.Scan(&thread.ID, &thread.Title, &thread.Author, &thread.Forum, &thread.Message, &thread.Votes, &thread.Slug, &thread.Created)

		got = append(got, *thread)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(got)
}

/*
 6666
66
66666
66  66
 6666
*/

func getPostDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["id"]

	postId, _ := strconv.Atoi(slug)

	var postFull PostFull

	sql := fmt.Sprintf("SELECT id, author, created, forum, isEdited, message, parent, thread FROM posts WHERE id = %d", postId)

	rows, err := db.Query(sql)

	if err != nil {
		http.Error(w, "Can't find post", 404)
		return
	}

	for rows.Next() {
		rows.Scan(&postFull.Post.ID, &postFull.Post.Author, &postFull.Post.Created, &postFull.Post.Forum, &postFull.Post.IsEdited, &postFull.Post.Message, &postFull.Post.Parent)
	}

	sql2 := fmt.Sprintf("SELECT nickname, fullname, about, email FROM users WHERE nickname = ''", postFull.Post.Author)

	rows2, err := db.Query(sql2)

	if err != nil {
		http.Error(w, "some error", 404)
		return
	}

	for rows2.Next() {
		rows2.Scan(&postFull.Author.Nickname, &postFull.Author.Fullname, &postFull.Author.About, &postFull.Author.Email)
	}

	sql3 := fmt.Sprintf("SELECT id, title, author, forum, message, votes, slug, created FROM threads WHERE id = %d", postFull.Post.Thread)

	rows3, err := db.Query(sql3)

	if err != nil {
		http.Error(w, "some error", 404)
		return
	}

	for rows3.Next() {
		rows3.Scan(&postFull.Thread.ID, &postFull.Thread.Title, &postFull.Thread.Author, &postFull.Thread.Forum, &postFull.Thread.Message, &postFull.Thread.Votes, &postFull.Thread.Slug, &postFull.Thread.Created)
	}

	sql4 := fmt.Sprintf("SELECT title, username, slug, posts, threads FROM forums WHERE slug = '%s'", postFull.Post.Forum)

	rows4, err := db.Query(sql4)

	if err != nil {
		http.Error(w, "some error", 404)
		return
	}

	for rows4.Next() {
		rows4.Scan(&postFull.Thread.ID, &postFull.Thread.Title, &postFull.Thread.Author, &postFull.Thread.Forum, &postFull.Thread.Message, &postFull.Thread.Votes, &postFull.Thread.Slug, &postFull.Thread.Created)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(postFull)
}

/*
777777
   77
  77
 77
77
*/

func setPost(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	vars := mux.Vars(r)
	slug := vars["id"]

	postId, _ := strconv.Atoi(slug)

	var postUpdate PostUpdate
	json.Unmarshal(reqBody, &postUpdate)

	sql := fmt.Sprintf("UPDATE posts set (message, isEdited) = ('%s', true) WHERE id = %d", postUpdate.Message, postId)

	_, err := db.Query(sql)

	if err != nil {
		http.Error(w, "Can't find post", 404)
		return
	}

	var post Post

	sql2 := fmt.Sprintf("SELECT id, author, created, forum, isEdited, message, parent, thread FROM posts WHERE id=%d", postId)

	rows, err := db.Query(sql2)

	for rows.Next() {
		rows.Scan(&post.ID, &post.Author, &post.Created, &post.Forum, &post.IsEdited, &post.Message, &post.Parent, &post.Thread)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(post)
}

/*
 8888
88  88
 8888
88  88
 8888
*/

func serviceClear(w http.ResponseWriter, r *http.Request) {
	sql := fmt.Sprintf("TRUNCATE users CASCADE")

	_, err := db.Query(sql)

	if err != nil {
		http.Error(w, "Some error", 404)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

/*
 9999
99  99
 99999
    99
 9999
*/

func serviceStatus(w http.ResponseWriter, r *http.Request) {
	var status Status

	sql := fmt.Sprintf("SELECT COUNT(id) from users")
	sql2 := fmt.Sprintf("SELECT COUNT(id) from threads")
	sql3 := fmt.Sprintf("SELECT COUNT(id) from posts")
	sql4 := fmt.Sprintf("SELECT COUNT(id) from forums")

	db.QueryRow(sql).Scan(&status.User)
	db.QueryRow(sql2).Scan(&status.Thread)
	db.QueryRow(sql3).Scan(&status.Post)
	db.QueryRow(sql4).Scan(&status.Forum)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

/*
1111   000000
  11   00  00
  11   00  00
  11   00  00
111111 000000
*/

func createPost(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	vars := mux.Vars(r)
	slug := vars["id"]

	threadId, err := strconv.Atoi(slug)

	var thread Thread
	var threadSql string

	if err != nil {
		threadSql = fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE slug='%s'", slug)
	} else {
		threadSql = fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE id=%d", threadId)
	}

	rowsThread, errThreads := db.Query(threadSql)

	if errThreads != nil {
		http.Error(w, "Can't find thread", 404)
		return
	}

	rowsThread.Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Slug, &thread.Title)

	var sql string

	var posts []Post
	json.Unmarshal(reqBody, &posts)

	var got []int32
	for i := range posts {
		var id int32

		sql = fmt.Sprintf("INSERT INTO posts (author, created, forum, parent, thread, message) VALUES ('%s','%s','%s',%d,%d,'%s') WHERE thread=%d RETURNING id", posts[i].Author, posts[i].Created, thread.Forum, posts[i].Parent, thread.ID, posts[i].Message, thread.ID)
		rows, err := db.Query(sql)

		if err != nil {
			fmt.Println(err)
			http.Error(w, "Exist", 409)
			return
		}

		rows.Scan(&id)

		got = append(got, id)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(got)
}

func handleRequests() { // РОУТЫ !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	myRouter := mux.NewRouter().StrictSlash(true)

	myRouter.HandleFunc("/api/forum/create", createForum).Methods("POST")
	myRouter.HandleFunc("/api/forum/{slug}/details", detailsForum)
	myRouter.HandleFunc("/api/forum/{slug}/create", forumThreadCreate).Methods("POST")
	myRouter.HandleFunc("/api/forum/{slug}/users", forumThreadUsers)
	myRouter.HandleFunc("/api/forum/{slug}/threads", getForumThreads)
	myRouter.HandleFunc("/api/post/{id}/details", getPostDetails)
	myRouter.HandleFunc("/api/post/{id}/details", setPost).Methods("POST")
	myRouter.HandleFunc("/api/service/clear", serviceClear).Methods("POST")
	myRouter.HandleFunc("/api/service/status", serviceStatus)
	myRouter.HandleFunc("/api/thread/{slug_or_id}/create", createPost)
	myRouter.HandleFunc("/api/thread/{slug_or_id}/create", createPost)
	log.Fatal(http.ListenAndServe(":10000", myRouter))
}

func main() {
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	handleRequests()
}
