package main

import (
	"github.com/Unknwon/paginater"
	"github.com/arschles/go-bindata-html-template"
	"github.com/goincremental/negroni-sessions"
	"github.com/gorilla/context"
	"net/http"
	"strconv"
	"time"
)

func getTemplate(r *http.Request, template_name string) *template.Template {
	funcMap := template.FuncMap{
		"paginate_url": func(page int) string {
			values := r.URL.Query()
			values.Del("page")
			values.Add("page", strconv.Itoa(page))
			r.URL.RawQuery = values.Encode()
			return r.URL.String()
		},
		"per_page_url": func(per_page int) string {
			values := r.URL.Query()
			values.Del("items_by_page")
			values.Add("items_by_page", strconv.Itoa(per_page))
			r.URL.RawQuery = values.Encode()
			return r.URL.String()
		},
		"getContextBool": func(key string) bool {
			return context.Get(r, key).(bool)
		},
	}
	t, err := template.New("mytmpl", Asset).Funcs(funcMap).ParseFiles(
		template_name,
		"templates/layout.html",
		"templates/includes/paginate.html",
	)
	checkErr(err)
	return t
}

func Index(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	session := sessions.GetSession(r)
	t := getTemplate(r, "templates/index.html")

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
	}

	items_by_page, err := strconv.Atoi(r.URL.Query().Get("items_by_page"))
	if err != nil {
		items_by_page = default_items_by_page
	}

	total_links := countLinks("")
	var result_total int

	var bms []*BookmarkItem
	search := r.URL.Query().Get("search")
	if search != "" {
		result_total, bms = searchBookmark(search, page, items_by_page)
	} else {
		bms = queryBookmark(page, items_by_page, r.URL.Query().Get("tags"))
		result_total = countLinks(r.URL.Query().Get("tags"))
	}

	data := struct {
		Bms         []*BookmarkItem
		Page        *paginater.Paginater
		TotalLinks  int
		ItemsByPage int
		Search      string
	}{
		Bms:         bms,
		Page:        paginater.New(result_total, items_by_page, page, 9),
		TotalLinks:  total_links,
		ItemsByPage: items_by_page,
		Search:      search,
	}

	context.Set(r, "index_page", true)
	if session.Get("login") != nil {
		context.Set(r, "login", true)
	}

	err = t.Execute(w, data)
}

func Edit(w http.ResponseWriter, r *http.Request, params map[string]string) {
	session := sessions.GetSession(r)
	t := getTemplate(r, "templates/edit.html")

	var bookmark_item BookmarkItem

	if _, ok := params["id"]; ok {
		id, err := strconv.ParseInt(params["id"], 10, 64)
		checkErr(err)

		bookmark_item = *getBookmark(id)
	} else {
		url := appendHttp(r.URL.Query().Get("url"))
		title, err := extractPageTitle(url)
		if err != nil {
			title = ""
		}
		bookmark_item = BookmarkItem{
			0,
			url,
			title,
			time.Time{},
			nil,
		}
	}

	data := struct {
		Item BookmarkItem
	}{
		Item: bookmark_item,
	}
	if session.Get("login") != nil {
		context.Set(r, "login", true)
	}

	err := t.Execute(w, data)
	checkErr(err)
}

func Save(w http.ResponseWriter, r *http.Request, params map[string]string) {
	session := sessions.GetSession(r)
	if session.Get("login") == nil {
		http.Redirect(w, r, "../../", 303)
		return
	}

	var link_id int64
	var err error
	if _, ok := params["id"]; ok {
		link_id, err = strconv.ParseInt(params["id"], 10, 64)
		checkErr(err)

		updateLink(
			link_id,
			r.FormValue("title"),
			appendHttp(r.FormValue("url")),
			r.FormValue("tags"),
		)
	} else {
		link_id = insertLink(
			r.FormValue("title"),
			appendHttp(r.FormValue("url")),
			r.FormValue("tags"),
		)
	}
	bookmark_item := getBookmark(link_id)
	indexBookmarkItem(bookmark_item)

	http.Redirect(w, r, "../../", 303)
}

func Delete(w http.ResponseWriter, r *http.Request, params map[string]string) {
	session := sessions.GetSession(r)
	if session.Get("login") == nil {
		http.Redirect(w, r, "../../", 303)
		return
	}

	stmt, err := DB.Prepare("DELETE FROM links WHERE id=?")
	checkErr(err)

	_, err = stmt.Exec(params["id"])
	checkErr(err)

	http.Redirect(w, r, "../../", 303)
}

func LoginForm(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	session := sessions.GetSession(r)
	t := getTemplate(r, "templates/login.html")

	data := struct {
		Error string
	}{
		Error: "",
	}

	errors := session.Flashes("errors")
	if len(errors) > 0 {
		data.Error = errors[0].(string)
	}

	t.Execute(w, data)
}

func Login(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	session := sessions.GetSession(r)
	if r.FormValue("password") == Password {
		session.Set("login", true)
		http.Redirect(w, r, "../", 303)
	} else {
		session.AddFlash("Password invalid", "errors")
		http.Redirect(w, r, ".", 303)
	}
}

func Logout(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	session := sessions.GetSession(r)
	session.Delete("login")
	http.Redirect(w, r, "../", 303)
}

func FetchTitle(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	url := appendHttp(r.URL.Query().Get("url"))
	title, err := extractPageTitle(url)
	if err != nil {
		title = ""
	}
	w.Write([]byte(title))
}
