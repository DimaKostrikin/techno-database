package main

import (
	"context"
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
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/lib/pq"
)

var ctx = context.Background()

var db, err = pgxpool.Connect(ctx, "postgres://yourname:yourpassword@localhost:5432/postgre")

var getUsersLimit = `SELECT nickname, fullname, email, about FROM usersForums JOIN users ON (usersForums.username = users.nickname) WHERE usersForums.forum = $1 ORDER BY users.nickname COLLATE "ucs_basic" ASC LIMIT $2`
var getUsersLimitDesc = `SELECT nickname, fullname, email, about FROM usersForums JOIN users ON (usersForums.username = users.nickname) WHERE usersForums.forum = $1 ORDER BY users.nickname COLLATE "ucs_basic" DESC LIMIT $2`
var getUsersLimitSince = `SELECT nickname, fullname, email, about FROM usersForums JOIN users ON (usersForums.username = users.nickname) WHERE usersForums.forum = $1 AND users.nickname > $2 COLLATE "ucs_basic" ORDER BY users.nickname COLLATE "ucs_basic" ASC LIMIT $3`
var getUsersLimitSinceDesc = `SELECT nickname, fullname, email, about FROM usersForums JOIN users ON (usersForums.username = users.nickname) WHERE usersForums.forum = $1 AND users.nickname < $2 COLLATE "ucs_basic" ORDER BY users.nickname COLLATE "ucs_basic" DESC LIMIT $3`

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
	tx, _ := db.Begin(ctx)
	defer tx.Commit(ctx)

	reqBody, _ := ioutil.ReadAll(r.Body)

	var forum Forum
	json.Unmarshal(reqBody, &forum)

	var userName string
	tx.QueryRow(ctx, "SELECT nickname FROM users WHERE nickname=$1", forum.User).Scan(&userName)

	if userName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find user"

		json.NewEncoder(w).Encode(Error)
		return
	}

	_, err := tx.Exec(ctx, "INSERT INTO forums (username, slug, title) VALUES ($1,$2,$3)", userName, forum.Slug, forum.Title)
	forum.User = userName

	if err != nil {
		if strings.Contains(err.Error(), "insert or update") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find user"

			json.NewEncoder(w).Encode(Error)
			return
		}

		if strings.Contains(err.Error(), "duplicate key value") {
			var got Forum
			db.QueryRow(ctx, "SELECT title, username, slug, posts, threads FROM forums WHERE slug=$1", forum.Slug).Scan(&got.Title, &got.User, &got.Slug, &got.Posts, &got.Threads)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(got)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(forum)
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

	var forum Forum
	db.QueryRow(ctx, "SELECT title, username as user, slug, posts, threads FROM forums WHERE slug=$1", slug).Scan(&forum.Title, &forum.User, &forum.Slug, &forum.Posts, &forum.Threads)

	if forum.Slug == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find Forum"

		json.NewEncoder(w).Encode(Error)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(forum)
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
	timeNowStr := strfmt.DateTime(time.Now())

	var thread Thread
	json.Unmarshal(reqBody, &thread)
	var strFmt strfmt.DateTime
	if thread.Created == strFmt {
		thread.Created = timeNowStr
	}

	var forumName string

	db.QueryRow(ctx, "SELECT slug FROM forums WHERE slug=$1", slug).Scan(&forumName)

	if forumName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find forum"

		json.NewEncoder(w).Encode(Error)
		return
	}

	insertId := 0
	err := db.QueryRow(ctx, "INSERT INTO threads (author, forum, message, title, slug, created) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6) RETURNING id", thread.Author, forumName, thread.Message, thread.Title, thread.Slug, thread.Created).Scan(&insertId)

	if err != nil {
		if strings.Contains(err.Error(), "violates foreign key") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find author"

			json.NewEncoder(w).Encode(Error)
			return
		}

		if strings.Contains(err.Error(), "duplicate key value") {
			var threadOut Thread
			db.QueryRow(ctx, "SELECT id, author, forum, message, title, slug, created FROM threads WHERE slug=$1", thread.Slug).Scan(&threadOut.ID, &threadOut.Author, &threadOut.Forum, &threadOut.Message, &threadOut.Title, &threadOut.Slug, &threadOut.Created)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(threadOut)
			return
		}
	}

	thread.ID = int32(insertId)
	thread.Forum = forumName

	//db.Exec(ctx, "INSERT INTO usersForums (username, forum) VALUES ($1, $2)", thread.Author, slug)

	//db.Exec(ctx, "UPDATE forums SET threads = threads + 1 WHERE slug = $1", slug)

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

	var checkId int
	db.QueryRow(ctx, "SELECT id FROM forums WHERE slug=$1", slug).Scan(&checkId)
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

	limit := 1

	if limitString != "" {
		limit, _ = strconv.Atoi(limitString)
	}

	var desc bool = false

	if descString != "" {
		desc, _ = strconv.ParseBool(descString)
	}

	var rows pgx.Rows
	var err1 error

	if sinceString == "" {
		if desc == true {
			rows, err1 = db.Query(ctx, getUsersLimitDesc, slug, limit)
		} else {
			rows, err1 = db.Query(ctx, getUsersLimit, slug, limit)
		}
	} else {
		if desc == true {
			rows, err1 = db.Query(ctx, getUsersLimitSinceDesc, slug, sinceString, limit)
		} else {
			rows, err1 = db.Query(ctx, getUsersLimitSince, slug, sinceString, limit)
		}
	}

	if err1 != nil {
		fmt.Println(err1)
	}

	got := make([]User, 0)
	for rows.Next() {
		user := new(User)

		rows.Scan(&user.Nickname, &user.Fullname, &user.Email, &user.About)

		got = append(got, *user)
	}

	rows.Close()
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
	//timeNowStr := strfmt.DateTime(sinceString)
	sinceTime, _ := strfmt.ParseDateTime(sinceString)

	descString := queryParam.Get("desc")

	rowsCheck, _ := db.Query(ctx, "SELECT id FROM forums WHERE slug = $1", slug)
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

	sql := fmt.Sprintf("SELECT id, title, author, forum, message, votes, created, COALESCE(slug, '') as slug FROM threads WHERE forum = '%s'", slug)

	var desc bool = false
	if descString != "" {
		desc, _ = strconv.ParseBool(descString)
	}

	if sinceString != "" {
		if desc == true {
			sql = sql + fmt.Sprintf(" AND created <='%s'", sinceTime)
		} else {
			sql = sql + fmt.Sprintf(" AND created >='%s'", sinceTime)
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

	rows, _ := db.Query(ctx, sql)
	defer rows.Close()

	got := make([]Thread, 0)

	var countRows int
	for rows.Next() {
		var thread Thread

		rows.Scan(&thread.ID, &thread.Title, &thread.Author, &thread.Forum, &thread.Message, &thread.Votes, &thread.Created, &thread.Slug)

		got = append(got, thread)
		countRows++
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

	db.QueryRow(ctx, "SELECT id, author, created, forum, isEdited, message, parent, thread FROM posts WHERE id = $1", postId).Scan(&postFull.Post.ID, &postFull.Post.Author, &postFull.Post.Created, &postFull.Post.Forum, &postFull.Post.IsEdited, &postFull.Post.Message, &postFull.Post.Parent, &postFull.Post.Thread)

	if postFull.Post.ID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find post"

		json.NewEncoder(w).Encode(Error)
		return
	}

	var postSend PostFull

	if strings.Contains(relatedString, "user") {
		db.QueryRow(ctx, "SELECT nickname, fullname, about, email FROM users WHERE nickname = $1", postFull.Post.Author).Scan(&user1.Nickname, &user1.Fullname, &user1.About, &user1.Email)

		postSend.Author = postFull.Author
	}

	if strings.Contains(relatedString, "thread") {
		db.QueryRow(ctx, "SELECT id, title, author, forum, message, votes, created, slug FROM threads WHERE id = $1", postFull.Post.Thread).Scan(&thread1.ID, &thread1.Title, &thread1.Author, &thread1.Forum, &thread1.Message, &thread1.Votes, &thread1.Created, &thread1.Slug)

		postSend.Thread = postFull.Thread
	}

	if strings.Contains(relatedString, "forum") {

		db.QueryRow(ctx, "SELECT title, username, slug, posts, threads FROM forums WHERE slug = $1", postFull.Post.Forum).Scan(&forum1.Title, &forum1.User, &forum1.Slug, &forum1.Posts, &forum1.Threads)
		postSend.Forum = postFull.Forum
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	postSend.Post = postFull.Post

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

	db.QueryRow(ctx, "SELECT id, author, created, forum, isEdited, message, parent, thread FROM posts WHERE id=$1", postId).Scan(&post.ID, &post.Author, &post.Created, &post.Forum, &post.IsEdited, &post.Message, &post.Parent, &post.Thread)

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
		return
	}

	if postUpdate.Message == post.Message {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(post)
		return
	}

	db.Exec(ctx, "UPDATE posts set (message, isEdited) = ($1, true) WHERE id = $2", postUpdate.Message, postId)

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
	db.Exec(ctx, "TRUNCATE users CASCADE")

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

	sql := "SELECT COUNT(nickname) from users"
	sql2 := "SELECT COUNT(id) from threads"
	sql3 := "SELECT COUNT(id) from posts"
	sql4 := "SELECT COUNT(id) from forums"

	db.QueryRow(ctx, sql).Scan(&status.User)
	db.QueryRow(ctx, sql2).Scan(&status.Thread)
	db.QueryRow(ctx, sql3).Scan(&status.Post)
	db.QueryRow(ctx, sql4).Scan(&status.Forum)

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
	tx, _ := db.Begin(ctx)
	defer tx.Commit(ctx)

	reqBody, _ := ioutil.ReadAll(r.Body)
	vars := mux.Vars(r)
	slug := vars["slug_or_id"]

	timeNowStr := strfmt.DateTime(time.Now())

	threadId, err := strconv.Atoi(slug)

	var thread Thread
	var threadSql string

	if err != nil {
		threadSql = fmt.Sprintf("SELECT id, forum FROM threads WHERE slug='%s'", slug)
	} else {
		threadSql = fmt.Sprintf("SELECT id, forum FROM threads WHERE id=%d", threadId)
	}

	tx.QueryRow(ctx, threadSql).Scan(&thread.ID, &thread.Forum)

	if thread.ID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find thread"

		json.NewEncoder(w).Encode(Error)
		return
	}

	var posts []Post
	json.Unmarshal(reqBody, &posts)

	for i := range posts {

		var strFmt strfmt.DateTime
		if posts[i].Created == strFmt {
			posts[i].Created = timeNowStr
		}

		err := tx.QueryRow(ctx, "INSERT INTO posts (author, created, forum, parent, thread, message) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id", posts[i].Author, posts[i].Created, thread.Forum, posts[i].Parent, thread.ID, posts[i].Message).Scan(&posts[i].ID)

		if err != nil {
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

		posts[i].Thread = thread.ID
		posts[i].Forum = thread.Forum
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(posts)
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
		threadSql = fmt.Sprintf("SELECT id, author, created, forum, message, title, votes, slug FROM threads WHERE slug='%s'", slug)
	} else {
		threadSql = fmt.Sprintf("SELECT id, author, created, forum, message, title, votes, slug FROM threads WHERE id=%d", threadId)
	}

	db.QueryRow(ctx, threadSql).Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Title, &thread.Votes, &thread.Slug)

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
		selectThreadSql = fmt.Sprintf("SELECT id, author, created, forum, message, title, votes, slug FROM threads WHERE slug='%s'", slug)
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

		selectThreadSql = fmt.Sprintf("SELECT id, author, created, forum, message, title, votes, slug FROM threads WHERE id=%d", threadId)
	}

	_, err2 := db.Exec(ctx, threadSql)

	if err2 != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		var Error ErrorMessage
		Error.Message = "Cant find tHrEaD"

		json.NewEncoder(w).Encode(Error)
		return
	}

	db.QueryRow(ctx, selectThreadSql).Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Title, &thread.Votes, &thread.Slug)

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

		db.QueryRow(ctx, "SELECT id FROM threads WHERE slug=$1", slug).Scan(&threadIDGot)

		threadId = threadIDGot

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
		var threadCheckId int
		db.QueryRow(ctx, "SELECT id FROM threads WHERE id=$1", threadId).Scan(&threadCheckId)
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

	}

	posts := make([]Post, 0)

	//sql := fmt.Sprintf("SELECT id, parent, author, message, isEdited, forum, thread, created FROM posts WHERE thread=%d", threadId)

	rows, _ := db.Query(ctx, sqlString)
	defer rows.Close()

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
		db.QueryRow(ctx, "SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE slug=$1", slug).Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)

		if thread.ID == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find thread"

			json.NewEncoder(w).Encode(Error)
			return
		}

	} else {
		db.QueryRow(ctx, "SELECT id, author, created, forum, message, title, votes, slug FROM threads WHERE id=$1", threadId).Scan(&thread.ID, &thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Title, &thread.Votes, &thread.Slug)

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

	db.QueryRow(ctx, "SELECT id, username, voice FROM votes WHERE thread=$1 AND username=$2", thread.ID, vote.Nickname).Scan(&voteGot.Id, &voteGot.Nickname, &voteGot.Voice)

	if vote.Voice == -voteGot.Voice && vote.Nickname == voteGot.Nickname {
		thread.Votes = thread.Votes + 2*vote.Voice

		db.Exec(ctx, "UPDATE votes SET voice=$1 WHERE id=$2", vote.Voice, voteGot.Id)
		db.Exec(ctx, "UPDATE threads SET votes=$1 WHERE id=$2", thread.Votes, thread.ID)
	}

	if voteGot.Id == 0 {
		_, err := db.Exec(ctx, "INSERT INTO votes (username, voice, thread) VALUES ($1, $2, $3)", vote.Nickname, vote.Voice, thread.ID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			var Error ErrorMessage
			Error.Message = "Cant find user"

			json.NewEncoder(w).Encode(Error)
			return
		}

		thread.Votes = thread.Votes + vote.Voice
		db.Exec(ctx, "UPDATE threads SET votes=$1 WHERE id=$2", thread.Votes, thread.ID)
	}

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
	userGot.Nickname = nickname

	sql := fmt.Sprintf("INSERT INTO users (nickname, fullname, about, email) VALUES ('%s', '%s', '%s', '%s')", nickname, userGot.Fullname, userGot.About, userGot.Email)

	_, err1 := db.Exec(ctx, sql)

	if err1 != nil {
		var returnedUser []User
		sqlGet := fmt.Sprintf("SELECT nickname, fullname, about, email FROM users WHERE nickname='%s' OR email='%s'", nickname, userGot.Email)

		rowsGet, _ := db.Query(ctx, sqlGet)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(userGot)
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
	db.QueryRow(ctx, sqlGet).Scan(&returnedUser.Nickname, &returnedUser.Fullname, &returnedUser.About, &returnedUser.Email)

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

	sql := fmt.Sprintf("SELECT nickname, fullname, about, email FROM users WHERE nickname='%s'", nickname)

	var selectUser User
	db.QueryRow(ctx, sql).Scan(&selectUser.Nickname, &selectUser.Fullname, &selectUser.About, &selectUser.Email)

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

	_, err := db.Exec(ctx, sqlUpdate)

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

func handleRequests() { // ?????????? !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
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
	//db.Config().ConnConfig.PreferSimpleProtocol = true

	handleRequests()
}
