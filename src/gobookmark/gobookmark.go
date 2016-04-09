package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/cheggaaa/pb"
	"github.com/codegangsta/cli"
	"github.com/codegangsta/negroni"
	"github.com/dimfeld/httptreemux"
	"github.com/goincremental/negroni-sessions"
	"github.com/goincremental/negroni-sessions/cookiestore"
	"github.com/gorilla/context"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	Password string
)

func stringFlag(name, value, usage string, envvar string) cli.StringFlag {
	return cli.StringFlag{
		Name:   name,
		Value:  value,
		Usage:  usage,
		EnvVar: envvar,
	}
}

func openDatabases(filename string) {
	cwd, _ := os.Getwd()
	filename = path.Join(cwd, filename)

	db_filename := fmt.Sprintf("%s.db", filename)
	log.Printf("Use %s SqlLite database", db_filename)
	DB = openDatabase(db_filename)

	index_filename := fmt.Sprintf("%s.index", filename)
	log.Printf("Use %s Bleve database", index_filename)
	INDEX = openBleve(index_filename)
}

func resetDatabases(filename string) {
	cwd, _ := os.Getwd()
	filename = path.Join(cwd, filename)

	db_filename := fmt.Sprintf("%s.db", filename)
	log.Printf("Reset %s SqlLite database", db_filename)
	os.Remove(db_filename)

	index_filename := fmt.Sprintf("%s.index", filename)
	log.Printf("Reset %s Bleve database", index_filename)
	os.RemoveAll(index_filename)
}

const default_items_by_page = 25

func init() {
	Password = "password"
}

func GlobalVariableMiddleware(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	context.Set(r, "login", false)
	context.Set(r, "index_page", false)
	next(rw, r)
}

func initApp() *negroni.Negroni {
	router := httptreemux.New()
	router.GET("/", Index)
	router.GET("/add/", Edit)
	router.GET("/fetch-title/", FetchTitle)
	router.POST("/add/", Save)
	router.GET("/:id/delete/", Delete)
	router.GET("/:id/edit/", Edit)
	router.POST("/:id/edit/", Save)
	router.GET("/login/", LoginForm)
	router.POST("/login/", Login)
	router.GET("/logout/", Logout)

	n := negroni.Classic()

	store := cookiestore.New([]byte("secret123"))
	n.Use(sessions.Sessions("my_session", store))
	n.Use(negroni.HandlerFunc(GlobalVariableMiddleware))
	n.Use(negroni.NewStatic(
		&AssetFS{
			Asset:     Asset,
			AssetDir:  AssetDir,
			AssetInfo: AssetInfo,
			Prefix:    "public",
		},
	))

	n.UseHandler(router)
	return n
}

func importFile(filename string) {
	filename, err := absPath(filename)
	checkErr(err)
	f, err := os.Open(filename)
	checkErr(err)

	doc, err := goquery.NewDocumentFromReader(f)
	checkErr(err)
	items := doc.Find("DT > A")
	bar := pb.StartNew(len(items.Nodes))
	items.Each(func(i int, s *goquery.Selection) {
		bar.Increment()
		add_date_str, _ := s.Attr("add_date")
		add_date_int, err := strconv.ParseInt(add_date_str, 10, 64)
		checkErr(err)

		stmt, err := DB.Prepare("INSERT INTO links (title, url, createdate) VALUES(?, ?, ?)")
		checkErr(err)

		href, _ := s.Attr("href")

		bm := new(BookmarkItem)
		bm.Title = s.Text()
		bm.Url = href
		bm.CreateDate = time.Unix(add_date_int, 0)

		res, err := stmt.Exec(
			bm.Title,
			bm.Url,
			bm.CreateDate,
		)
		checkErr(err)

		bm.Id, err = res.LastInsertId()
		tags, _ := s.Attr("tags")
		updateLinksTags(bm.Id, strings.Split(tags, ","))

		// Index

		bm.Tags = getLinksTags(bm.Id)
		indexBookmarkItem(bm)
	})
	bar.FinishPrint("The End!")
}

func main() {
	app := cli.NewApp()
	app.Name = "gobookmark"
	app.Version = "0.1.0"
	app.Usage = "A personnal bookmark service"
	app.Flags = []cli.Flag{
		stringFlag("data, d", "gobookmark", "Database filename", "GOBOOKMARK_DATABASE"),
	}
	app.Commands = []cli.Command{
		{
			Name:  "web",
			Usage: "Start Gobookmark web server",
			Description: `Gobookmark web server is the only thing you need to run,
and it takes care of all the other things for you`,
			Flags: []cli.Flag{
				stringFlag("port, p", "8000", "Web server port", "GOBOOKMARK_PORT"),
				stringFlag("host", "localhost", "Web server host", "GOBOOKMARK_HOST"),
				stringFlag("password", "password", "Set login password", "GOBOOKMARK_PASSWORD"),
			},
			Action: func(c *cli.Context) {
				Password = c.String("password")
				openDatabases(c.Parent().String("data"))
				n := initApp()
				n.Run(fmt.Sprintf("%s:%s", c.String("host"), c.String("port")))
			},
		},
		{
			Name:  "import",
			Usage: "Import bookmark HTML file",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "reset, r",
					Usage: "Reset database before importation",
				},
			},
			ArgsUsage: "<input-file>",
			Action: func(c *cli.Context) {
				if len(c.Args()) == 0 {
					log.Print("Error : <input-file> missing")
				} else {
					if c.Bool("reset") {
						resetDatabases(c.Parent().String("data"))
					}
					openDatabases(c.Parent().String("data"))
					importFile(c.Args()[0])
				}
			},
		},
		{
			Name:  "reindex",
			Usage: "Execute plain text search indexation",
			Action: func(c *cli.Context) {
				openDatabases(c.Parent().String("data"))
				log.Print("Reindex database with Bleve")
				indexAllBookmark()
			},
		},
	}
	app.Run(os.Args)
}
