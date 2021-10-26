package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

var router = mux.NewRouter()
var db *sql.DB

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<h1>Hello, homepage :)</h1>")
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "此博客是用以记录编程笔记，如您有反馈或建议，请联系 "+
		"<a href=\"mailto:sigongzu@163.com\">sigongzu@163.com</a>")
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "<h1>Hello, not found :(</h1>")
}

func articlesShowHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	fmt.Fprint(w, "文章ID："+id)
}

func articlesIndexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "访问文章列表")
}

type ArticlesFormData struct {
	Title, Body string
	URL         *url.URL
	Errors      map[string]string
}

func articlesStoreHandler(w http.ResponseWriter, r *http.Request) {

	title := r.PostFormValue("title")
	body := r.PostFormValue("body")

	errors := make(map[string]string)

	if title == "" {
		errors["title"] = "标题不可空"
	} else if utf8.RuneCountInString(title) < 3 || utf8.RuneCountInString(title) > 40 {
		errors["title"] = "标题长度介于3-40"
	}

	if body == "" {
		errors["body"] = "内容不可空"
	} else if utf8.RuneCountInString(body) < 10 {
		errors["body"] = "内容要长度大于9"
	}

	if len(errors) == 0 {
		fmt.Fprint(w, "验证通过<br>")
		fmt.Fprintf(w, "%v<br>", title)
		fmt.Fprintf(w, "%v<br>", utf8.RuneCountInString(title))
		fmt.Fprintf(w, "%v<br>", body)
		fmt.Fprintf(w, "%v<br>", utf8.RuneCountInString(body))
	} else {
		// html := `
		// <!DOCTYPE html>
		// <html lang="en">
		// <head>
		// 	<title>创建文章 —— 我的技术博客</title>
		// 	<style type="text/css">.error {color: red;}</style>
		// </head>
		// <body>
		// 	<form action="{{ .URL }}" method="post">
		// 		<p><input type="text" name="title" value="{{ .Title }}"></p>
		// 		{{ with .Errors.title }}
		// 		<p class="error">{{ . }}</p>
		// 		{{ end }}
		// 		<p><textarea name="body" cols="30" rows="10">{{ .Body }}</textarea></p>
		// 		{{ with .Errors.body }}
		// 		<p class="error">{{ . }}</p>
		// 		{{ end }}
		// 		<p><button type="submit">提交</button></p>
		// 	</form>
		// </body>
		// </html>
		// `
		storeURL, _ := router.Get("articles.store").URL()

		data := ArticlesFormData{
			Title:  title,
			Body:   body,
			URL:    storeURL,
			Errors: errors,
		}

		// tmpl, err := template.New("create-form").Parse(html)
		tmpl, err := template.ParseFiles("resources/views/articles/create.gohtml")
		if err != nil {
			panic(err)
		}

		tmpl.Execute(w, data)
	}
}

func forceHTMLMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func removeTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		}

		next.ServeHTTP(w, r)
	})
}

func articlesCreateHandler(w http.ResponseWriter, r *http.Request) {
	// html := `
	// <!DOCTYPE html>
	// <html lang="en">
	// <head>
	// 	<title>创建文章 —— 我的技术博客</title>
	// </head>
	// <body>
	// 	<form action="%s?test=data" method="post">
	// 		<p><input type="text" name="title"></p>
	// 		<p><textarea name="body" cols="30" rows="10"></textarea></p>
	// 		<p><button type="submit">提交</button></p>
	// 	</form>
	// </body>
	// </html>
	// `
	storeURL, _ := router.Get("articles.store").URL()
	data := ArticlesFormData{
		Title:  "",
		Body:   "",
		URL:    storeURL,
		Errors: nil,
	}
	tmpl, err := template.ParseFiles("resources/views/articles/create.gohtml")
	if err != nil {
		panic(err)
	}

	tmpl.Execute(w, data)
}

func initDB() {
	var err error
	config := mysql.Config{
		User:                 "root",
		Passwd:               "jkdf1212",
		Addr:                 "127.0.0.1:3306",
		Net:                  "tcp",
		DBName:               "gitgoblog",
		AllowNativePasswords: true,
	}

	db, err = sql.Open("mysql", config.FormatDSN())
	checkError(err)

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	err = db.Ping()
	checkError(err)
}

func checkError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func createTables() {
	createArticlesSQL := `
	create table if not exists articles(
		id bigint(20) primary key auto_increment not null,
		title varchar(255) collate utf8mb4_unicode_ci not null,
		body longtext collate utf8mb4_unicode_ci
	);
	`

	_, err := db.Exec(createArticlesSQL)
	checkError(err)
}

func main() {

	initDB()
	createTables()

	router.HandleFunc("/", homeHandler).Methods("GET").Name("home")
	router.HandleFunc("/about", aboutHandler).Methods("GET").Name("about")

	router.HandleFunc("/articles/{id:[0-9]+}", articlesShowHandler).Methods("GET").Name("articles.show")
	router.HandleFunc("/articles", articlesIndexHandler).Methods("GET").Name("articles.index")
	router.HandleFunc("/articles", articlesStoreHandler).Methods("POST").Name("articles.store")
	router.HandleFunc("/articles/create", articlesCreateHandler).Methods("GET").Name("articles.create")

	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	router.Use(forceHTMLMiddleware)

	http.ListenAndServe(":3000", removeTrailingSlash(router))
}
