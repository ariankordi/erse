package main

import (
	"database/sql"
	"gopkg.in/gorp.v1"
	_ "github.com/mattn/go-sqlite3"
	"encoding/json"

	"github.com/valyala/fasthttp"
	"github.com/fasthttp-contrib/websocket"
	"github.com/buaazp/fasthttprouter"
	"strconv"
	"fmt"
	"log"
	//"time"
)

func initDb() *gorp.DbMap {
	db, _ := sql.Open("sqlite3", "./thing.db")
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	return dbmap
}
var dbmap = initDb()

type Place struct {
	Id          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
	Image       string `db:"image" json:"image"`
	PostCount   int64  `db:"-" json:"count"`
}

type Post struct {
	Id      int64  `db:"id" json:"id"`
	Name    string `db:"name" json:"name"`
	Ip      string `db:"ip" json:"ip,omitempty"`
	Place   int64  `db:"place" json:"-"`
	Message string `db:"message" json:"message"`
}

type PlaceAndPosts struct {
	Place Place  `json:"place"`
	Posts []Post `json:"messages"`
}

func Index(ctx *fasthttp.RequestCtx) {
     fasthttp.ServeFile(ctx, "./index.html")
}

func ListPlaces(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")

	var places []Place
	_, _ = dbmap.Select(&places, "select * from place order by id desc limit 20")
	for i, place := range places {
		msgCount, _ := dbmap.SelectInt("select count(id) from messages where place = ?", strconv.FormatInt(place.Id, 10))
		places[i].PostCount = msgCount
	}
	output, _ := json.Marshal(places)
	fmt.Fprint(ctx, string(output))
}

func ListPosts(ctx *fasthttp.RequestCtx) {
	placeId := string(ctx.QueryArgs().Peek("id"))
	if placeId == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	var placeAndPosts PlaceAndPosts
	dbmap.SelectOne(&placeAndPosts.Place, "select * from place where id = ? limit 1", placeId)
	dbmap.Select(&placeAndPosts.Posts, "select id, place, name, message from messages where place = ? order by id desc limit 50", placeId)
	output, _ := json.Marshal(placeAndPosts)
	fmt.Fprint(ctx, string(output))
}

func MakePost(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")

	placeId := string(ctx.QueryArgs().Peek("id"))
	if placeId == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	Name := string(ctx.PostArgs().Peek("name"))
	Message := string(ctx.PostArgs().Peek("message"))
	if Message == "" || Name == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	last, _ := dbmap.Exec("insert into messages(place, name, message, ip) values(?, ?, ?, ?)", placeId, Name, Message, ctx.RemoteAddr().String())
	lastI, _ := last.LastInsertId()
	lastId := strconv.FormatInt(lastI, 10)
	fmt.Fprintf(ctx, "{\"id\": " + lastId + "}")
	placeId64, _ := strconv.ParseInt(placeId, 10, 64)
	createdPost := Post{Id: lastI, Name: Name, Message: Message, Place: placeId64}
	postChan <- createdPost
}

var upgrader = websocket.New(postReceiver)
var PlaceId int64
func PostServiceUpgrade(ctx *fasthttp.RequestCtx) {
	placeId := string(ctx.QueryArgs().Peek("id"))
	if placeId == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	PlaceId, _ = strconv.ParseInt(placeId, 10, 64)
	upgrader.Upgrade(ctx)
}

type Client struct {
	Dead  bool
	Place int64
	Chan  chan string
}

var postChan = make(chan Post, 1)
var serverQueue = make(chan Client, 4)
func postQueueServer(serverQueue chan Client) {
	var clients []Client
	for {
		select {
			case client, _ := <-serverQueue:
				clients = append(clients, client)
			case post, _ := <-postChan:
				for _, c := range clients {
						if c.Place == post.Place {
							output, _ := json.Marshal(post)
							c.Chan <- string(output)
					}
				}
			}
		}
}

func postReceiver(c *websocket.Conn) {
	defer c.Close()
	client := Client{Place: PlaceId}
	client.Chan = make(chan string, 1)
	serverQueue <- client
	for {
		select {
			case text, _ := <-client.Chan:
					writer, err := c.NextWriter(websocket.TextMessage)
					if err == nil {
						writer.Write([]byte(text))
						writer.Close()
					}
			}
		}
}

func main() {
	go postQueueServer(serverQueue)
	router := fasthttprouter.New()
	router.GET("/api/places.jsp", ListPlaces)
	router.GET("/api/posts.jsp", ListPosts)
	router.POST("/api/createPost.jsp", MakePost)
	router.GET("/api/postService.jsp", PostServiceUpgrade)

		 // Static files
		 fs := &fasthttp.FS{
			 Root: "./static",
			 AcceptByteRange: true,
			 Compress: true,
			 PathRewrite: fasthttp.NewPathSlashesStripper(1),
		 }
		 fsHandler := fs.NewRequestHandler()
		 router.GET("/static/*path", fsHandler)
     router.NotFound = func(ctx *fasthttp.RequestCtx) {
			 // Vue HTML5 index
			 Index(ctx)
			 //ctx.SetStatusCode(fasthttp.StatusNotFound)
			 //fmt.Fprintf(ctx, "Not found")
     }

	fmt.Println("Running server!")
	log.Fatal(fasthttp.ListenAndServe(":8024", router.Handler))
}
