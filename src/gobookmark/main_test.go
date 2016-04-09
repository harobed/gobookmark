package main

import (
	"bytes"
	"database/sql"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func openTestDatabase() *sql.DB {
	const test_database = "gobookmark-test.db"
	const test_bleve = "gobookmark-test.index"

	if _, err := os.Stat(test_database); err == nil {
		os.Remove(test_database)
	}

	if _, err := os.Stat(test_bleve); err == nil {
		os.RemoveAll(test_bleve)
	}

	INDEX = openBleve(test_bleve)

	return openDatabase(test_database)
}

func TestIndex(t *testing.T) {
	DB = openTestDatabase()
	defer DB.Close()
	app := initApp()
	server := httptest.NewServer(app)
	defer server.Close()

	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookieJar,
	}

	resp, _ := client.Get(server.URL + "/")
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestLogin(t *testing.T) {
	DB = openTestDatabase()
	defer DB.Close()
	app := initApp()
	server := httptest.NewServer(app)
	defer server.Close()

	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookieJar,
	}

	resp, _ := client.Get(server.URL + "/login/")
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assertResponseBodyContains(t, resp, "Password")
}

func assertResponseBodyContains(t *testing.T, resp *http.Response, contains string) {
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	assert.Contains(t, string(bodyBytes), contains)
}

func assertResponseBodyNotContains(t *testing.T, resp *http.Response, contains string) {
	body, _ := ioutil.ReadAll(resp.Body)

	assert.NotContains(t, string(body), contains)
}

func TestAddBookmark(t *testing.T) {
	DB = openTestDatabase()
	defer DB.Close()
	app := initApp()
	server := httptest.NewServer(app)
	defer server.Close()

	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookieJar,
	}

	resp, _ := client.Get(server.URL + "/add/?url=http://cv.stephane-klein.info")
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assertResponseBodyContains(t, resp, "Développeur")
}

func TestSaveNewBookmark(t *testing.T) {
	DB = openTestDatabase()
	defer DB.Close()
	app := initApp()
	server := httptest.NewServer(app)
	defer server.Close()

	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookieJar,
	}
	client.PostForm(
		server.URL+"/login/",
		url.Values{
			"password": {"password"},
		},
	)

	resp, _ := client.PostForm(
		server.URL+"/add/",
		url.Values{
			"url":   {"http://cv.stephane-klein.info"},
			"title": {"Le CV de Stéphane Klein"},
			"tags":  {"tag1,tag2,tag3"},
		},
	)

	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assert.Equal(t, resp.Request.URL.Path, "/")

	assertResponseBodyContains(t, resp, "stephane")
	assertResponseBodyContains(t, resp, "tag2")
}

func TestDeleteBookmark(t *testing.T) {
	DB = openTestDatabase()
	defer DB.Close()
	app := initApp()
	server := httptest.NewServer(app)
	defer server.Close()

	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookieJar,
	}
	client.PostForm(
		server.URL+"/login/",
		url.Values{
			"password": {"password"},
		},
	)

	insertLink("Le Curriculum vitae de Stéphane Klein", "http://cv.stephane-klein.info", "")

	resp, _ := client.Get(server.URL + "/1/delete/")

	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assert.Equal(t, resp.Request.URL.Path, "/")

	assertResponseBodyNotContains(t, resp, "Curriculum")
}

func TestEditBookmark(t *testing.T) {
	DB = openTestDatabase()
	defer DB.Close()
	app := initApp()
	server := httptest.NewServer(app)
	defer server.Close()

	cookieJar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookieJar,
	}
	client.PostForm(
		server.URL+"/login/",
		url.Values{
			"password": {"password"},
		},
	)

	insertLink("Le CV de Stéphane Klein", "http://cv.stephane-klein.info", "")

	resp, _ := client.Get(server.URL + "/1/edit/")

	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assertResponseBodyContains(t, resp, "stephane")

	resp, _ = client.PostForm(
		server.URL+"/1/edit/",
		url.Values{
			"url":   {"http://cv.noelie-deschamps.info"},
			"title": {"Le CV de Noëlie Deschamps"},
		},
	)

	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assertResponseBodyContains(t, resp, "noelie")
}

func TestExtractPageTitle(t *testing.T) {
	title, _ := extractPageTitle("http://cv.stephane-klein.info")
	assert.Equal(t, title, "Curriculum vitæ de Stéphane Klein | CV | Développeur, Administrateur Système | 15 ans d'expérience")

	title, _ = extractPageTitle("https://golang.org/")
	assert.Equal(t, title, "The Go Programming Language")
}

func TestSearchByTags(t *testing.T) {
	DB = openTestDatabase()
	defer DB.Close()
	app := initApp()
	server := httptest.NewServer(app)
	defer server.Close()

	insertLink("AAAAAAAA", "http://example1.com", "python")
	insertLink("BBBBBBBB", "http://example2.com", "python")
	insertLink("CCCCCCCC", "http://example3.com", "python,golang")
	insertLink("DDDDDDDD", "http://example4.com", "python,golang")
	insertLink("EEEEEEEE", "http://example5.com", "golang")
	insertLink("FFFFFFFF", "http://example6.com", "golang")
	indexAllBookmark()

	total, _ := searchBookmark("[python]", 1, 10)
	assert.Equal(t, total, 4)

	total, _ = searchBookmark("[golang]", 1, 10)
	assert.Equal(t, total, 4)

	total, _ = searchBookmark("[golang][python]", 1, 10)
	assert.Equal(t, total, 2)

	total, bms := searchBookmark("[python] BBBBBBBB", 1, 10)
	assert.Equal(t, total, 4)
	assert.Equal(t, bms[0].Title, "BBBBBBBB")
}
