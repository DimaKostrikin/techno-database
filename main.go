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

	"time"

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
	Posts   int    `json:"posts, ommitempty"`
	Threads int    `json:"threads, ommitempty"`
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
	Created  strfmt.DateTime `json:"created"`
}

type PostFull struct {
	Post   Post    `json:"post"`
	Author *User   `json:"author,omitempty"`
	Thread *Thread `json:"thread,omitempty"`
	Forum  *Forum  `json:"forum,omitempty"`
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

type UpdateThread struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

type Vote struct {
	Id       int64  `json:id`
	Nickname string `json:"nickname"`
	Voice    int32  `json:"voice"`
	Thread   int32  `json:"thread"`
}

type ErrorMessage struct {
	Message string `json:"message"`
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

	sqlStringUserName := fmt.Sprintf("SELECT nickname FROM users WHERE nickname='%s'", forum.User)

	rowsUser, _ := db.Query(sqlStringUserName)
	defer rowsUser.Close()

	var userName string
	for rowsUser.Next() {
		rowsUser.Scan(&userName)
	}

	if userName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find user"

		json.NewEncoder(w).Encode(Error)
		return
	}

	sqlString := fmt.Sprintf("INSERT INTO forums (username, slug, title) VALUES ('%s','%s','%s')", userName, forum.Slug, forum.Title)
	//sqlString := "INSERT INTO forums (username, slug, title) VALUES ('abc','abcde','abc')"
	rowsToClose, err := db.Query(sqlString)
	if err == nil {
		defer rowsToClose.Close()
	}

	if err != nil {
		fmt.Println(err)
		if strings.Contains(err.Error(), "insert or update") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find user"

			json.NewEncoder(w).Encode(Error)
			return
		}

		if strings.Contains(err.Error(), "duplicate key value") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
		}
	}

	sqlString2 := fmt.Sprintf("SELECT title, username, slug, posts, threads FROM forums WHERE slug='%s'", forum.Slug)
	rows, _ := db.Query(sqlString2)
	defer rows.Close()

	var got Forum
	for rows.Next() {
		rows.Scan(&got.Title, &got.User, &got.Slug, &got.Posts, &got.Threads)
		fmt.Println(got.User)
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
	defer rows.Close()

	var got Forum
	for rows.Next() {
		forum := new(Forum)

		rows.Scan(&forum.Title, &forum.User, &forum.Slug, &forum.Posts, &forum.Threads)

		got = *forum
	}

	if got.Slug == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find Forum"

		json.NewEncoder(w).Encode(Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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

	sqlForumName := fmt.Sprintf("SELECT slug FROM forums WHERE slug='%s'", slug)
	rowsForumName, _ := db.Query(sqlForumName)
	defer rowsForumName.Close()
	var forumName string
	for rowsForumName.Next() {
		rowsForumName.Scan(&forumName)
	}
	if forumName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find forum"

		json.NewEncoder(w).Encode(Error)
		return
	}

	sql := fmt.Sprintf("INSERT INTO threads (author, forum, message, title, slug, created) VALUES ('%s', '%s', '%s', '%s', NULLIF('%s', ''), '%s') RETURNING id", thread.Author, forumName, thread.Message, thread.Title, thread.Slug, thread.Created)

	insertId := 0
	err := db.QueryRow(sql).Scan(&insertId)

	if err != nil {
		fmt.Println(err)
		if strings.Contains(err.Error(), "violates foreign key") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find author"

			json.NewEncoder(w).Encode(Error)
			return
		}

		if strings.Contains(err.Error(), "duplicate key value") {
			sqlOut := fmt.Sprintf("SELECT id, author, forum, message, title, slug, created FROM threads WHERE slug='%s'", thread.Slug)

			var threadOut Thread
			db.QueryRow(sqlOut).Scan(&threadOut.ID, &threadOut.Author, &threadOut.Forum, &threadOut.Message, &threadOut.Title, &threadOut.Slug, &threadOut.Created)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(threadOut)
			return
		}
	}

	sql2 := fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE id=%d", insertId)
	db.QueryRow(sql2).Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)

	sql3 := fmt.Sprintf("INSERT INTO usersForums (username, forum) VALUES ('%s','%s')", thread.Author, slug)
	db.Exec(sql3)

	sql4 := fmt.Sprintf("UPDATE forums SET threads = threads + 1 WHERE slug = '%s'", slug)
	db.Exec(sql4)

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

	queryParam := r.URL.Query()

	sqlCheck := fmt.Sprintf("SELECT id FROM forums WHERE slug='%s'", slug)
	var checkId int
	db.QueryRow(sqlCheck).Scan(&checkId)
	if checkId == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find forum"

		json.NewEncoder(w).Encode(Error)
		return
	}

	limitString := queryParam.Get("limit")
	sinceString := queryParam.Get("since")
	descString := queryParam.Get("desc")

	var desc bool = false

	if descString != "" {
		desc, _ = strconv.ParseBool(descString)
	}

	sql := fmt.Sprintf("SELECT nickname, fullname, email, about FROM usersForums JOIN users ON (usersForums.username = users.nickname) WHERE usersForums.forum = '%s'", slug)

	if sinceString != "" {
		if desc == true {
			sql = sql + fmt.Sprintf(`AND nickname < '%s' COLLATE "ucs_basic"`, sinceString)
		} else {
			sql = sql + fmt.Sprintf(`AND nickname > '%s' COLLATE "ucs_basic"`, sinceString)
		}
	}

	if desc == true {
		sql = sql + ` ORDER BY LOWER(nickname) COLLATE "ucs_basic" DESC`
	} else {
		sql = sql + ` ORDER BY LOWER(nickname) COLLATE "ucs_basic" ASC`
	}

	if limitString != "" {
		limit, _ := strconv.Atoi(limitString)

		sql = sql + fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(sql)
	defer rows.Close()

	if err != nil {
		fmt.Print(err)
	}

	got := make([]User, 0)
	for rows.Next() {
		user := new(User)

		rows.Scan(&user.Nickname, &user.Fullname, &user.Email, &user.About)

		got = append(got, *user)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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

	queryParam := r.URL.Query()

	limitString := queryParam.Get("limit")
	sinceString := queryParam.Get("since")
	descString := queryParam.Get("desc")

	sqlCheck := fmt.Sprintf("SELECT id FROM forums WHERE slug = '%s'", slug)
	rowsCheck, _ := db.Query(sqlCheck)
	defer rowsCheck.Close()
	var checkId int
	for rowsCheck.Next() {
		rowsCheck.Scan(&checkId)
	}
	if checkId == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find forum"

		json.NewEncoder(w).Encode(Error)
		return
	}

	sql := fmt.Sprintf("SELECT id, title, author, forum, message, votes, slug, created FROM threads WHERE forum = '%s'", slug)

	var desc bool = false
	if descString != "" {
		desc, _ = strconv.ParseBool(descString)
	}

	if sinceString != "" {
		if desc == true {
			sql = sql + fmt.Sprintf(" AND created <='%s'", sinceString)
		} else {
			sql = sql + fmt.Sprintf(" AND created >='%s'", sinceString)
		}
	}

	if desc == true {
		sql = sql + " ORDER BY created DESC"
	} else {
		sql = sql + " ORDER BY created ASC"
	}

	if limitString != "" {
		limit, _ := strconv.Atoi(limitString)

		sql = sql + fmt.Sprintf(" LIMIT %d", limit)
	}

	fmt.Printf(sql)

	rows, err := db.Query(sql)
	defer rows.Close()

	if err != nil {
		fmt.Print(err)
	}

	got := make([]Thread, 0)
	for rows.Next() {
		var thread Thread

		rows.Scan(&thread.ID, &thread.Title, &thread.Author, &thread.Forum, &thread.Message, &thread.Votes, &thread.Slug, &thread.Created)

		got = append(got, thread)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

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

	queryParam := r.URL.Query()

	relatedString := queryParam.Get("related")

	var postFull PostFull

	var user1 User
	postFull.Author = &user1

	var forum1 Forum
	postFull.Forum = &forum1

	var thread1 Thread
	postFull.Thread = &thread1

	sql := fmt.Sprintf("SELECT id, author, created, forum, isEdited, message, parent, thread FROM posts WHERE id = %d", postId)

	rows, err := db.Query(sql)

	if err == nil {
		defer rows.Close()
	}

	if err != nil {
		fmt.Println(err)
		http.Error(w, "Can't find post", 404)
		return
	}

	for rows.Next() {
		rows.Scan(&postFull.Post.ID, &postFull.Post.Author, &postFull.Post.Created, &postFull.Post.Forum, &postFull.Post.IsEdited, &postFull.Post.Message, &postFull.Post.Parent, &postFull.Post.Thread)
	}

	if postFull.Post.ID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find post"

		json.NewEncoder(w).Encode(Error)
		return
	}

	sql2 := fmt.Sprintf("SELECT nickname, fullname, about, email FROM users WHERE nickname = '%s'", postFull.Post.Author)

	rows2, err := db.Query(sql2)
	if err == nil {
		defer rows2.Close()
	}

	if err != nil {
		fmt.Println(err)
		http.Error(w, "some error", 404)
		return
	}

	for rows2.Next() {
		rows2.Scan(&user1.Nickname, &user1.Fullname, &user1.About, &user1.Email)
	}

	sql3 := fmt.Sprintf("SELECT id, title, author, forum, message, votes, slug, created FROM threads WHERE id = %d", postFull.Post.Thread)

	rows3, err := db.Query(sql3)
	if err == nil {
		defer rows3.Close()
	}

	if err != nil {
		fmt.Println(err)
		http.Error(w, "some error", 404)
		return
	}

	for rows3.Next() {
		rows3.Scan(&thread1.ID, &thread1.Title, &thread1.Author, &thread1.Forum, &thread1.Message, &thread1.Votes, &thread1.Slug, &thread1.Created)
	}

	sql4 := fmt.Sprintf("SELECT title, username, slug, posts, threads FROM forums WHERE slug = '%s'", postFull.Post.Forum)

	rows4, err := db.Query(sql4)
	defer rows4.Close()
	if err == nil {
		defer rows4.Close()
	}

	if err != nil {
		fmt.Println(err)
		http.Error(w, "some error", 404)
		return
	}

	for rows4.Next() {
		rows4.Scan(&forum1.Title, &forum1.User, &forum1.Slug, &forum1.Posts, &forum1.Threads)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var postSend PostFull
	postSend.Post = postFull.Post
	fmt.Println(relatedString)

	if strings.Contains(relatedString, "user") {
		postSend.Author = postFull.Author
	}

	if strings.Contains(relatedString, "thread") {
		postSend.Thread = postFull.Thread
	}

	if strings.Contains(relatedString, "forum") {
		postSend.Forum = postFull.Forum
	}

	json.NewEncoder(w).Encode(postSend)
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

	var post Post

	sql2 := fmt.Sprintf("SELECT id, author, created, forum, isEdited, message, parent, thread FROM posts WHERE id=%d", postId)

	rows, err := db.Query(sql2)
	defer rows.Close()

	for rows.Next() {
		rows.Scan(&post.ID, &post.Author, &post.Created, &post.Forum, &post.IsEdited, &post.Message, &post.Parent, &post.Thread)
	}

	if post.ID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find post"

		json.NewEncoder(w).Encode(Error)
		return
	}

	if postUpdate.Message == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(post)

		rows.Close()
		return
	}

	if postUpdate.Message == post.Message {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(post)

		rows.Close()
		return
	}

	sql := fmt.Sprintf("UPDATE posts set (message, isEdited) = ('%s', true) WHERE id = %d", postUpdate.Message, postId)

	rowsToClose, err := db.Query(sql)

	if err == nil {
		defer rowsToClose.Close()
	}

	if err != nil {
		http.Error(w, "Can't find post", 404)
		return
	}

	post.Message = postUpdate.Message
	post.IsEdited = true
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

	raa, err := db.Query(sql)
	defer raa.Close()

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

	sql := fmt.Sprintf("SELECT COUNT(nickname) from users")
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
	slug := vars["slug_or_id"]

	timeNowStr := strfmt.DateTime(time.Now())

	threadId, err := strconv.Atoi(slug)

	var thread Thread
	var threadSql string

	if err != nil {
		threadSql = fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE slug='%s'", slug)
	} else {
		threadSql = fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE id=%d", threadId)
	}

	fmt.Println(threadSql)

	rowsThread, err2 := db.Query(threadSql)
	defer rowsThread.Close()
	if err2 != nil {
		fmt.Println(err2)
	}

	for rowsThread.Next() {
		rowsThread.Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)
	}

	if thread.ID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find thread"

		json.NewEncoder(w).Encode(Error)
		return
	}

	var sql string

	var posts []Post
	json.Unmarshal(reqBody, &posts)

	var got []int32
	for i := range posts {
		var id int32

		var strFmt strfmt.DateTime
		if posts[i].Created == strFmt {
			posts[i].Created = timeNowStr
		}

		sql = fmt.Sprintf("INSERT INTO posts (author, created, forum, parent, thread, message) VALUES ('%s','%s','%s',%d,%d,'%s') RETURNING id", posts[i].Author, posts[i].Created, thread.Forum, posts[i].Parent, thread.ID, posts[i].Message)
		rows, err := db.Query(sql)

		if err != nil {
			fmt.Println(err)
			if strings.Contains(err.Error(), "insert or update") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				var Error ErrorMessage
				Error.Message = "User not found"

				json.NewEncoder(w).Encode(Error)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			var Error ErrorMessage
			Error.Message = "Error with post"

			json.NewEncoder(w).Encode(Error)
			return
		}

		for rows.Next() {
			rows.Scan(&id)
		}

		got = append(got, id)
		sql3 := fmt.Sprintf("INSERT INTO usersForums (username, forum) VALUES ('%s','%s')", posts[i].Author, thread.Forum)
		db.Exec(sql3)
		rows.Close()
	}

	gotPosts := make([]Post, 0)
	for i := range got {
		var post Post

		sql = fmt.Sprintf("SELECT id, parent, author, message, isEdited, forum, thread, created FROM posts WHERE id=%d", got[i])

		rows, _ := db.Query(sql)

		for rows.Next() {
			rows.Scan(&post.ID, &post.Parent, &post.Author, &post.Message, &post.IsEdited, &post.Forum, &post.Thread, &post.Created)
		}

		gotPosts = append(gotPosts, post)

		rows.Close() // Мб смысла нет бтв да да
	}

	db.Exec(fmt.Sprintf("UPDATE forums SET posts = posts + %d WHERE slug='%s'", len(gotPosts), thread.Forum))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(gotPosts)
}

/*
1111   1111
  11     11
  11     11
  11     11
111111 111111
*/

func detailsThread(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug_or_id"]

	threadId, err := strconv.Atoi(slug)

	var thread Thread
	var threadSql string

	if err != nil {
		threadSql = fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE slug='%s'", slug)
	} else {
		threadSql = fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE id=%d", threadId)
	}

	rowsThread, _ := db.Query(threadSql)

	defer rowsThread.Close()

	for rowsThread.Next() {
		rowsThread.Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)
	}

	if thread.ID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find THREAD"

		json.NewEncoder(w).Encode(Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(thread)
}

/*
1111   222222
  11        2
  11   222222
  11   2
111111 222222
*/

func updateDetailsThread(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	vars := mux.Vars(r)
	slug := vars["slug_or_id"]

	var updateThread UpdateThread
	json.Unmarshal(reqBody, &updateThread)

	threadId, err := strconv.Atoi(slug)

	var thread Thread

	var threadSql string
	var selectThreadSql string

	if err != nil {
		if updateThread.Message == "" && updateThread.Title == "" {
			threadSql = ""
		}
		if updateThread.Message != "" && updateThread.Title == "" {
			threadSql = fmt.Sprintf("UPDATE threads SET message = '%s' WHERE slug='%s'", updateThread.Message, slug)
		}
		if updateThread.Message == "" && updateThread.Title != "" {
			threadSql = fmt.Sprintf("UPDATE threads SET title = '%s' WHERE slug='%s'", updateThread.Title, slug)
		}
		if updateThread.Message != "" && updateThread.Title != "" {
			threadSql = fmt.Sprintf("UPDATE threads SET (title, message) = ('%s', '%s') WHERE slug='%s'", updateThread.Title, updateThread.Message, slug)
		}
		selectThreadSql = fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE slug='%s'", slug)
	} else {
		if updateThread.Message == "" && updateThread.Title == "" {
			threadSql = ""
		}
		if updateThread.Message != "" && updateThread.Title == "" {
			threadSql = fmt.Sprintf("UPDATE threads SET message = '%s' WHERE id=%d", updateThread.Message, threadId)
		}
		if updateThread.Message == "" && updateThread.Title != "" {
			threadSql = fmt.Sprintf("UPDATE threads SET title = '%s' WHERE id=%d", updateThread.Title, threadId)
		}
		if updateThread.Message != "" && updateThread.Title != "" {
			threadSql = fmt.Sprintf("UPDATE threads SET (title, message) = ('%s', '%s') WHERE id=%d", updateThread.Title, updateThread.Message, threadId)
		}

		selectThreadSql = fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE id=%d", threadId)
	}

	rowsToClose, err2 := db.Query(threadSql)
	if err2 == nil {
		defer rowsToClose.Close()
	}

	if err2 != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find tHrEaD"

		json.NewEncoder(w).Encode(Error)
		return
	}

	rowsThread, _ := db.Query(selectThreadSql)
	defer rowsThread.Close()

	for rowsThread.Next() {
		rowsThread.Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)
	}

	if thread.ID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find THREEAD"

		json.NewEncoder(w).Encode(Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(thread)
}

/*
1111   333333
  11       33
  11   333333
  11       33
111111 333333
*/

func getThreadPosts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	slug := vars["slug_or_id"]
	threadId, err := strconv.Atoi(slug)

	queryParam := r.URL.Query()

	limitString := queryParam.Get("limit")
	sinceString := queryParam.Get("since")
	descString := queryParam.Get("desc")
	sortString := queryParam.Get("sort")

	if sortString == "" {
		sortString = "flat"
	}

	var desc bool = false
	if descString != "" {
		desc, _ = strconv.ParseBool(descString)
	}

	if err != nil {
		var threadIDGot int

		sql := fmt.Sprintf("SELECT id FROM threads WHERE slug='%s'", slug)
		rows, _ := db.Query(sql)

		for rows.Next() {
			rows.Scan(&threadIDGot)
		}
		threadId = threadIDGot
		rows.Close()

		if threadIDGot == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find thread"

			json.NewEncoder(w).Encode(Error)
			return
		}
	}

	if err == nil {
		sql2 := fmt.Sprintf("SELECT id FROM threads WHERE id=%d", threadId)
		var threadCheckId int
		db.QueryRow(sql2).Scan(&threadCheckId)
		if threadCheckId == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find thread"

			json.NewEncoder(w).Encode(Error)
			return
		}
	}

	var sqlString string
	if sortString == "flat" {
		sqlString = fmt.Sprintf("SELECT id, parent, author, message, isEdited, forum, thread, created FROM posts WHERE thread=%d", threadId)

		if sinceString != "" {
			since, _ := strconv.Atoi(sinceString)

			if desc == true {
				sqlString = sqlString + fmt.Sprintf(" AND id<%d", since)
			} else {
				sqlString = sqlString + fmt.Sprintf(" AND id>%d", since)
			}
		}

		if desc == true {
			sqlString = sqlString + " ORDER BY id DESC"
		} else {
			sqlString = sqlString + " ORDER BY id ASC"
		}

		if limitString != "" {
			limit, _ := strconv.Atoi(limitString)

			sqlString = sqlString + fmt.Sprintf(" LIMIT %d", limit)
		}

		fmt.Println(sqlString)
	}

	if sortString == "tree" {
		sqlString = fmt.Sprintf("SELECT id, parent, author, message, isEdited, forum, thread, created FROM posts WHERE thread=%d", threadId)

		if sinceString != "" {
			since, _ := strconv.Atoi(sinceString)

			if desc == true {
				sqlString = sqlString + fmt.Sprintf(" AND path < (SELECT path FROM posts WHERE id=%d)", since)
			} else {
				sqlString = sqlString + fmt.Sprintf(" AND path > (SELECT path FROM posts WHERE id=%d)", since)
			}
		}

		if desc == true {
			sqlString = sqlString + " ORDER BY path DESC"
		} else {
			sqlString = sqlString + " ORDER BY path ASC"
		}

		if limitString != "" {
			limit, _ := strconv.Atoi(limitString)

			sqlString = sqlString + fmt.Sprintf(" LIMIT %d", limit)
		}
	}

	if sortString == "parent_tree" {
		sqlString = fmt.Sprintf("SELECT path FROM posts r WHERE r.parent=0 AND r.thread=%d", threadId)

		if sinceString != "" {
			since, _ := strconv.Atoi(sinceString)

			if desc == true {
				sqlString = sqlString + fmt.Sprintf(" AND r.path[1] < (SELECT path[1] FROM posts WHERE id=%d)", since)
			} else {
				sqlString = sqlString + fmt.Sprintf(" AND r.path[1] > (SELECT path[1] FROM posts WHERE id=%d)", since)
			}
		}

		if desc == true {
			sqlString = sqlString + " ORDER BY r.path DESC"
		} else {
			sqlString = sqlString + " ORDER BY r.path ASC"
		}

		if limitString != "" {
			limit, _ := strconv.Atoi(limitString)

			sqlString = sqlString + fmt.Sprintf(" LIMIT %d", limit)
		}

		wrapperString := "WITH sub AS ("
		wrapperString = wrapperString + sqlString + ")" + " SELECT p.id, p.parent, p.author, p.message, p.isEdited, p.forum, p.thread, p.created FROM posts p JOIN sub ON sub.path[1] = p.path[1]"

		if desc == true {
			wrapperString = wrapperString + " ORDER BY p.path[1] DESC, p.path"
		} else {
			wrapperString = wrapperString + " ORDER BY p.path[1] ASC, p.path"
		}

		sqlString = wrapperString

		fmt.Println(sqlString)
	}

	posts := make([]Post, 0)

	//sql := fmt.Sprintf("SELECT id, parent, author, message, isEdited, forum, thread, created FROM posts WHERE thread=%d", threadId)

	rows, err2 := db.Query(sqlString)
	defer rows.Close()

	if err2 != nil {
		fmt.Println(err2)
	}

	for rows.Next() {
		var post Post

		rows.Scan(&post.ID, &post.Parent, &post.Author, &post.Message, &post.IsEdited, &post.Forum, &post.Thread, &post.Created)
		if post.ID == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find post"

			json.NewEncoder(w).Encode(Error)
			return
		}

		posts = append(posts, post)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(posts)
}

/*
1111   44  44
  11   44  44
  11   444444
  11       44
111111     44
*/

func threadVote(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	vars := mux.Vars(r)
	slug := vars["slug_or_id"]

	var vote Vote
	json.Unmarshal(reqBody, &vote)

	threadId, err := strconv.Atoi(slug)

	var thread Thread

	if err != nil {
		sql := fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE slug='%s'", slug)
		rows, err1 := db.Query(sql)

		if err1 != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find thread"

			json.NewEncoder(w).Encode(Error)
			return
		}

		for rows.Next() {
			rows.Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)
		}

		rows.Close()

		if thread.ID == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find thread"

			json.NewEncoder(w).Encode(Error)
			return
		}

	} else {
		sql := fmt.Sprintf("SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE id=%d", threadId)

		rows, _ := db.Query(sql)

		for rows.Next() {
			rows.Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)
		}
		rows.Close()

		if thread.ID == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find thread"

			json.NewEncoder(w).Encode(Error)
			return
		}
	}

	var voteGot Vote

	fmt.Println(thread.ID)

	sqlVote := fmt.Sprintf("SELECT id, username, voice FROM votes WHERE thread=%d AND username='%s'", thread.ID, vote.Nickname)

	rowsVote, err2 := db.Query(sqlVote)
	defer rowsVote.Close()

	for rowsVote.Next() {
		fmt.Println(err2)
		rowsVote.Scan(&voteGot.Id, &voteGot.Nickname, &voteGot.Voice)
		//fmt.Println(fmt.Sprintf("UPDATING VOTEGOT %s", voteGot.Nickname))
	}

	if voteGot.Id == 0 {
		sqlCheck := fmt.Sprintf("SELECT nickname FROM users WHERE nickname='%s'", vote.Nickname)
		var checkString string
		db.QueryRow(sqlCheck).Scan(&checkString)
		if checkString == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find user"

			json.NewEncoder(w).Encode(Error)
			return
		}

		sqlVote := fmt.Sprintf("INSERT INTO votes (username, voice, thread) VALUES ('%s', %d, %d)", vote.Nickname, vote.Voice, thread.ID)

		rows, err := db.Query(sqlVote)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find user"

			json.NewEncoder(w).Encode(Error)
			return
		}
		thread.Votes += vote.Voice
		rows.Close()
	} else {
		if voteGot.Voice != vote.Voice {
			sqlVote := fmt.Sprintf("UPDATE votes SET voice=%d WHERE id=%d", vote.Voice, voteGot.Id)
			thread.Votes += 2 * vote.Voice

			db.Exec(sqlVote)
		}
	}

	/*
		if vote.Voice == -voteGot.Voice && vote.Nickname == voteGot.Nickname {
			fmt.Println("YA EST")
			sqlVote := fmt.Sprintf("UPDATE votes SET voice=%d WHERE id=%d", vote.Voice, voteGot.Id)
			thread.Votes = thread.Votes + 2*vote.Voice
			sqlUpdateTable := fmt.Sprintf("UPDATE threads SET votes=%d WHERE id=%d", thread.Votes, thread.ID)
			db.Exec(sqlVote)
			db.Exec(sqlUpdateTable)
		}

		if voteGot.Nickname == "" {
			sqlVote := fmt.Sprintf("INSERT INTO votes (username, voice, thread) VALUES ('%s', %d, %d)", vote.Nickname, vote.Voice, thread.ID)

			rows, err := db.Query(sqlVote)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				var Error ErrorMessage
				Error.Message = "Cant find user"

				json.NewEncoder(w).Encode(Error)
				return
			}

			thread.Votes = thread.Votes + vote.Voice
			sqlUpdateTable := fmt.Sprintf("UPDATE threads SET votes=%d WHERE id=%d", thread.Votes, thread.ID)

			db.Exec(sqlUpdateTable)

			rows.Close()
		}
	*/

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(thread)
}

/*
1111   555555
  11   55
  11   555555
  11       55
111111 555555
*/

func createUser(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	vars := mux.Vars(r)
	nickname := vars["nickname"]

	var userGot User
	json.Unmarshal(reqBody, &userGot)

	sql := fmt.Sprintf("INSERT INTO users (nickname, fullname, about, email) VALUES ('%s', '%s', '%s', '%s')", nickname, userGot.Fullname, userGot.About, userGot.Email)

	rowsToClose, err1 := db.Query(sql)
	if err1 == nil {
		defer rowsToClose.Close()
	}

	if err1 != nil {
		var returnedUser []User
		fmt.Println(err1)
		sqlGet := fmt.Sprintf("SELECT nickname, fullname, about, email FROM users WHERE nickname='%s' OR email='%s'", nickname, userGot.Email)

		rowsGet, _ := db.Query(sqlGet)
		defer rowsGet.Close()

		for rowsGet.Next() {
			var us User
			rowsGet.Scan(&us.Nickname, &us.Fullname, &us.About, &us.Email)
			returnedUser = append(returnedUser, us)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(returnedUser)
		return
	}

	var returnedUser User
	sqlGet := fmt.Sprintf("SELECT nickname, fullname, about, email FROM users WHERE nickname='%s'", nickname)
	rowsGet, err2 := db.Query(sqlGet)
	defer rowsGet.Close()

	if err2 != nil {
		fmt.Println(err2)
	}

	for rowsGet.Next() {
		rowsGet.Scan(&returnedUser.Nickname, &returnedUser.Fullname, &returnedUser.About, &returnedUser.Email)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(returnedUser)
}

/*
1111   666666
  11   66
  11   666666
  11   66  66
111111 666666
*/

func userInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nickname := vars["nickname"]

	var returnedUser User
	sqlGet := fmt.Sprintf("SELECT nickname, fullname, about, email FROM users WHERE nickname='%s'", nickname)
	rowsGet, _ := db.Query(sqlGet)
	defer rowsGet.Close()

	for rowsGet.Next() {
		rowsGet.Scan(&returnedUser.Nickname, &returnedUser.Fullname, &returnedUser.About, &returnedUser.Email)
	}

	if returnedUser.Nickname == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find user"

		json.NewEncoder(w).Encode(Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(returnedUser)
}

/*
1111   777777
  11       77
  11       77
  11       77
111111     77
*/

func userChange(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	vars := mux.Vars(r)
	nickname := vars["nickname"]

	var userGot User
	json.Unmarshal(reqBody, &userGot)
	fmt.Print(userGot.About)

	sql := fmt.Sprintf("SELECT nickname, fullname, about, email FROM users WHERE nickname='%s'", nickname)

	rows, _ := db.Query(sql)
	defer rows.Close()

	var selectUser User
	for rows.Next() {
		rows.Scan(&selectUser.Nickname, &selectUser.Fullname, &selectUser.About, &selectUser.Email)
	}

	if selectUser.Nickname == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find user"

		json.NewEncoder(w).Encode(Error)
		return
	}

	userGot.Nickname = selectUser.Nickname

	if userGot.Fullname == "" {
		userGot.Fullname = selectUser.Fullname
	}

	if userGot.About == "" {
		userGot.About = selectUser.About
	}

	if userGot.Email == "" {
		userGot.Email = selectUser.Email
	}

	sqlUpdate := fmt.Sprintf("UPDATE users SET (fullname, about, email) = ('%s', '%s', '%s') WHERE nickname='%s'", userGot.Fullname, userGot.About, userGot.Email, userGot.Nickname)

	rowsClose, err := db.Query(sqlUpdate)

	if err == nil {
		defer rowsClose.Close()
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		var Error ErrorMessage
		Error.Message = "Updating email exist"

		json.NewEncoder(w).Encode(Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(userGot)
}

func handleRequests() { // РОУТЫ !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.Queries("query")

	myRouter.HandleFunc("/api/forum/create", createForum).Methods("POST")
	myRouter.HandleFunc("/api/forum/{slug}/details", detailsForum)
	myRouter.HandleFunc("/api/forum/{slug}/create", forumThreadCreate).Methods("POST")
	myRouter.HandleFunc("/api/forum/{slug}/users", forumThreadUsers)
	myRouter.HandleFunc("/api/forum/{slug}/threads", getForumThreads).Methods("GET")
	myRouter.HandleFunc("/api/post/{id}/details", getPostDetails).Methods("GET")
	myRouter.HandleFunc("/api/post/{id}/details", setPost).Methods("POST")
	myRouter.HandleFunc("/api/service/clear", serviceClear).Methods("POST")
	myRouter.HandleFunc("/api/service/status", serviceStatus)
	myRouter.HandleFunc("/api/thread/{slug_or_id}/create", createPost)
	myRouter.HandleFunc("/api/thread/{slug_or_id}/details", detailsThread).Methods("GET")
	myRouter.HandleFunc("/api/thread/{slug_or_id}/details", updateDetailsThread).Methods("POST")
	myRouter.HandleFunc("/api/thread/{slug_or_id}/posts", getThreadPosts)
	myRouter.HandleFunc("/api/thread/{slug_or_id}/vote", threadVote).Methods("POST")
	myRouter.HandleFunc("/api/user/{nickname}/create", createUser).Methods("POST")
	myRouter.HandleFunc("/api/user/{nickname}/profile", userInfo).Methods("GET")
	myRouter.HandleFunc("/api/user/{nickname}/profile", userChange).Methods("POST")
	log.Fatal(http.ListenAndServe(":5000", myRouter))
}

func main() {
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	handleRequests()
}
