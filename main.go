package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"log"
	"time"
	"fmt"
)

type Room struct {
	ID        string `gorm:"primary_key"`
	CreatedAt time.Time
	Replies   []Reply
	IsLogin   bool   `gorm:"-"`
	Password  []byte
}

type Session struct {
	ID        string
	CreatedAt time.Time
	Room      Room
	RoomID    string
}

type Reply struct {
	gorm.Model
	Room    Room
	RoomID  string
	Content string
	User string
}

var db *gorm.DB

func index(c *gin.Context) {
	room := getRoom(c)
	if !room.IsLogin {
		c.Redirect(303, "/login")
		return
	}
	var replies []Reply
	if err := db.Where("room_id = ?", room.ID).Order("created_at desc").Limit(20).Find(&replies).Error; err != nil {
		c.Error(err)
		return
	}
	page := struct {
		Room    Room
		Replies []Reply
	}{room, replies}
	c.HTML(200, "index.gohtml", page)
}

func post(c *gin.Context) {
	room := getRoom(c)
	if !room.IsLogin {
		c.String(401, "未登入")
		return
	}
	r := Reply{
		RoomID:  room.ID,
		Content: c.PostForm("reply"),
		User: c.PostForm("user"),
	}
	log.Println(r)
	if err := db.Create(&r).Error; err != nil {
		log.Panic(err)
		c.String(500, err.Error())
		return
	}
	c.String(200, fmt.Sprintf(`%s <%s> %s`, short(r.CreatedAt), r.User, r.Content))
	return
}

func short(t time.Time) string {
	return t.Format("15:04")
}

func init() {
	var err error
	db, err = gorm.Open("sqlite3", "data/data.sqlite")
	if err != nil {
		log.Fatal("Open database fatal:", err)
	}
	db.AutoMigrate(&Room{}, &Session{}, &Reply{})
}

func main() {
	r := gin.Default()
	funcmap := make(map[string]interface{})
	funcmap["short"] = short
	r.SetFuncMap(funcmap)
	r.LoadHTMLGlob("./templates/*.gohtml")
	r.GET("/", index)
	r.POST("/", post)
	r.GET("/login", func(c *gin.Context) {
		if getRoom(c).IsLogin {
			c.Redirect(303, "/")
		}
		page := struct {
			Title  string
			Action string
		}{"登入", "/login"}
		c.HTML(200, "login.gohtml", page)
	})
	r.POST("/login", login)
	r.GET("/create", func(c *gin.Context) {
		if getRoom(c).IsLogin {
			c.Redirect(303, "/")
		}
		page := struct {
			Title  string
			Action string
		}{"註冊", "/create"}
		c.HTML(200, "login.gohtml", page)
	})
	r.POST("/create", create)
	r.GET("/logout", logout)
	r.Run()
}
