package main

import (
	"database/sql"
	"github.com/blevesearch/bleve"
	"github.com/extemporalgenome/slug"
	_ "github.com/mattes/migrate/driver/sqlite3"
	"github.com/mattes/migrate/file"
	"github.com/mattes/migrate/migrate"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	DB    *sql.DB
	INDEX bleve.Index
)

type Tag struct {
	Id    int64
	Title string
	Slug  string
}

type BookmarkItem struct {
	Id         int64
	Url        string
	Title      string
	CreateDate time.Time
	Tags       []*Tag
}

func countLinks(tags string) int {
	var count int
	if tags != "" {
		err := DB.QueryRow(
			`SELECT
				COUNT(links.id)
			FROM
				links
			LEFT JOIN
				rel_links_tags
			ON
				rel_links_tags.link_id = links.id
			LEFT JOIN
				tags
			ON
				rel_links_tags.tag_id = tags.id
			WHERE
				tags.slug IN (?)`, tags).Scan(&count)
		checkErr(err)
	} else {
		err := DB.QueryRow("SELECT COUNT(id) FROM links").Scan(&count)
		checkErr(err)
	}
	return count
}

func openDatabase(filename string) *sql.DB {
	migrate.NonGraceful()
	migrate.UseStore(file.AssetStore{
		Asset:    Asset,
		AssetDir: AssetDir,
	})
	errors, ok := migrate.UpSync("sqlite3://"+filename, "migrations")
	if !ok {
		log.Fatalf("%v", errors)
	}

	db, err := sql.Open("sqlite3", filename)
	checkErr(err)

	return db
}

func indexBookmarkItem(item *BookmarkItem) error {
	x := struct {
		Id    int64  `json:"id"`
		Url   string `json:"url"`
		Title string `json:"title"`
		Tags  string `json:"tags"`
	}{
		Id:    item.Id,
		Url:   item.Url,
		Title: item.Title,
		Tags:  "",
	}
	for _, t := range item.Tags {
		x.Tags = x.Tags + " " + t.Slug
	}
	return INDEX.Index(strconv.FormatInt(item.Id, 10), x)
}

func indexAllBookmark() {
	rows, err := DB.Query("SELECT id, title, url, createdate FROM links")
	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		bm := new(BookmarkItem)

		err := rows.Scan(&bm.Id, &bm.Title, &bm.Url, &bm.CreateDate)
		checkErr(err)

		bm.Tags = getLinksTags(bm.Id)

		indexBookmarkItem(bm)
	}
}

func openBleve(filename string) (index bleve.Index) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		indexMapping := bleve.NewIndexMapping()
		linkMapping := bleve.NewDocumentMapping()

		linkTitleFieldMapping := bleve.NewTextFieldMapping()
		linkTitleFieldMapping.Analyzer = "en"
		linkMapping.AddFieldMappingsAt("title", linkTitleFieldMapping)

		linkUrlFieldMapping := bleve.NewTextFieldMapping()
		linkMapping.AddFieldMappingsAt("url", linkUrlFieldMapping)

		linkTagsFieldMapping := bleve.NewTextFieldMapping()
		linkMapping.AddFieldMappingsAt("tags", linkTagsFieldMapping)

		indexMapping.AddDocumentMapping("link", linkMapping)

		index, err = bleve.New(filename, indexMapping)
		checkErr(err)
	} else {
		index, err = bleve.Open(filename)
		checkErr(err)
	}

	return index
}

func getOrCreateTag(tag_name string) int64 {
	var id int64
	stmt, err := DB.Prepare("SELECT id FROM tags WHERE title=?")
	checkErr(err)

	rows, err := stmt.Query(tag_name)
	defer rows.Close()
	if rows.Next() {
		rows.Scan(&id)
	} else {
		stmt, err = DB.Prepare("INSERT INTO tags (title, slug) VALUES(?, ?)")
		checkErr(err)
		res, err := stmt.Exec(tag_name, slug.Slug(tag_name))
		id, err = res.LastInsertId()
		checkErr(err)
	}
	return id
}

func insertLink(title string, url string, tags string) (id int64) {
	stmt, err := DB.Prepare("INSERT INTO links (title, url) VALUES(?, ?)")
	checkErr(err)

	res, err := stmt.Exec(title, url)
	checkErr(err)
	link_id, err := res.LastInsertId()
	checkErr(err)
	updateLinksTags(link_id, strings.Split(tags, ","))
	return link_id
}

func updateLink(id int64, title string, url string, tags string) {
	stmt, err := DB.Prepare("UPDATE links SET title=?, url=? WHERE id=?")
	checkErr(err)

	_, err = stmt.Exec(title, url, id)

	updateLinksTags(id, strings.Split(tags, ","))
}

func updateLinksTags(link_id int64, tag_name_list []string) {
	stmt, err := DB.Prepare("DELETE FROM rel_links_tags WHERE link_id=?")
	checkErr(err)
	_, err = stmt.Exec(link_id)
	checkErr(err)

	for _, tag_name := range tag_name_list {
		tag_id := getOrCreateTag(tag_name)
		stmt, err = DB.Prepare("INSERT INTO rel_links_tags (link_id, tag_id) VALUES(?, ?)")
		_, err = stmt.Exec(link_id, tag_id)
		checkErr(err)
	}
}

func getLinksTags(link_id int64) (result []*Tag) {
	stmt, err := DB.Prepare(
		`SELECT
			tags.id,
			tags.title,
			tags.slug
		FROM
			rel_links_tags
		LEFT JOIN
			tags
		ON
			rel_links_tags.tag_id = tags.id
		WHERE
			rel_links_tags.link_id=?`)
	checkErr(err)
	rows, err := stmt.Query(link_id)
	defer rows.Close()
	checkErr(err)
	for rows.Next() {
		tag := new(Tag)
		err := rows.Scan(&tag.Id, &tag.Title, &tag.Slug)
		checkErr(err)
		result = append(result, tag)
	}
	return result
}

func getBookmark(id int64) *BookmarkItem {
	bookmark_item := new(BookmarkItem)
	err := DB.QueryRow("SELECT id, title, url, createdate FROM links WHERE id=?", id).Scan(
		&bookmark_item.Id,
		&bookmark_item.Title,
		&bookmark_item.Url,
		&bookmark_item.CreateDate,
	)
	checkErr(err)
	bookmark_item.Tags = getLinksTags(id)
	return bookmark_item
}

func queryBookmark(page int, items_by_page int, tags string) []*BookmarkItem {
	var rows *sql.Rows

	if tags != "" {
		stmt, err := DB.Prepare(
			`SELECT
				links.id,
				links.title,
				links.url,
				links.createdate
			FROM
				links
			LEFT JOIN
				rel_links_tags
			ON
				rel_links_tags.link_id = links.id
			LEFT JOIN
				tags
			ON
				rel_links_tags.tag_id = tags.id
			WHERE
				tags.slug IN (?)
			ORDER BY
				links.createdate DESC
			LIMIT ? OFFSET ?`)
		checkErr(err)
		rows, err = stmt.Query(tags, items_by_page, (page-1)*items_by_page)
		checkErr(err)
	} else {
		stmt, err := DB.Prepare(
			`SELECT
				id,
				title,
				url,
				createdate
			FROM
				links
			ORDER BY
				createdate DESC
			LIMIT ? OFFSET ?`)
		checkErr(err)
		rows, err = stmt.Query(items_by_page, (page-1)*items_by_page)
		checkErr(err)
	}
	defer rows.Close()

	bms := make([]*BookmarkItem, 0)
	for rows.Next() {
		bm := new(BookmarkItem)
		err := rows.Scan(&bm.Id, &bm.Title, &bm.Url, &bm.CreateDate)
		checkErr(err)

		bm.Tags = getLinksTags(bm.Id)

		bms = append(bms, bm)
	}
	return bms
}

func searchBookmark(search string, page int, items_by_page int) (total int, bms []*BookmarkItem) {
	tags := extractTags(search)
	search = removeTags(search)

	var tags_query_list []bleve.Query
	tags_query_list = nil

	if len(tags) > 0 {
		tags_query_list = make([]bleve.Query, 0)
		for _, tag := range tags {
			q := bleve.NewQueryStringQuery("tags:" + tag)
			tags_query_list = append(tags_query_list, q)
		}
	}

	var query bleve.Query

	if search != "" {
		query_list := make([]bleve.Query, 2)
		fuzzy_query := bleve.NewFuzzyQuery(search)
		fuzzy_query.FuzzinessVal = 1
		query_list[0] = fuzzy_query
		query_list[1] = bleve.NewRegexpQuery("[a-zA-Z0-9_]*" + search + "[a-zA-Z0-9_]*")
		query = bleve.NewBooleanQuery(tags_query_list, query_list, nil)
	} else {
		query = bleve.NewBooleanQuery(tags_query_list, nil, nil)
	}
	searchRequest := bleve.NewSearchRequestOptions(query, items_by_page, (page-1)*items_by_page, false)
	sr, err := INDEX.Search(searchRequest)
	checkErr(err)

	bms = make([]*BookmarkItem, 0)

	if sr.Total > 0 {
		if sr.Request.Size > 0 {
			for _, hit := range sr.Hits {
				id, err := strconv.ParseInt(hit.ID, 10, 64)
				checkErr(err)
				bms = append(bms, getBookmark(id))
			}
		}
	}

	return int(sr.Total), bms
}
