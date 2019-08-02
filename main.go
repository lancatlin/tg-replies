package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"time"
)

type Room struct {
	ID        string `form:"room_name" gorm:"primary_key"`
	CreatedAt time.Time
	Replies   []Reply
	IsLogin   bool   `gorm:"-"`
	Password  []byte `form:"password"`
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
}

var db *gorm.DB

func getRoom(c *gin.Context) Room {
	// 從 session 中取得 room
	sessionID, err := c.Cookie("session")
	if err != nil {
		return Room{IsLogin: false}
	}
	var room Room
	var session Session
	if err = db.Where(&Session{ID: sessionID}).First(&session).Error; err != nil {
		http.SetCookie(c.Writer, &http.Cookie{
			Name:   "session",
			MaxAge: -1,
		})
		log.Println(err)
		return Room{IsLogin: false}
	}
	if err = db.Model(&session).Related(&room).Error; err != nil {
		c.Error(err)
		return Room{IsLogin: false}
	}
	room.IsLogin = true
	return room
}

func login(c *gin.Context) {
	if getRoom(c).IsLogin {
		c.Redirect(303, "/")
		return
	}
	roomID := c.PostForm("room_id")
	password := c.PostForm("password")
	var room Room
	if err := db.Where(Room{ID: roomID}).First(&room).Error; gorm.IsRecordNotFoundError(err) {
		// 登入失敗
		c.String(401, "登入失敗：房間或密碼錯誤")
		return
	}
	log.Println(room)
	if err := bcrypt.CompareHashAndPassword(room.Password, []byte(password)); err != nil {
		// 密碼錯誤，登入失敗
		c.String(401, "登入失敗：房間或密碼錯誤")
		return
	}
	session := Session{
		ID:     uuid.NewV4().String(),
		RoomID: room.ID,
	}
	if err := db.Create(&session).Error; err != nil {
		c.Error(err)
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:  "session",
		Value: session.ID,
	})
	c.Redirect(303, "/")
}

func create(c *gin.Context) {
	var room Room
	if getRoom(c).IsLogin {
		c.Redirect(303, "/")
		return
	}
	roomID := c.PostForm("room_id")
	password := c.PostForm("password")
	if err := db.Where(Room{ID: roomID}).First(&room).Error; room.ID != "" {
		// 房間名已經使用過
		c.String(401, "創建失敗：房間名已經被使用過")
		return
	} else if err.Error() != "record not found" {
		c.Error(err)
		return
	}
	room.ID = roomID
	var err error
	room.Password, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.Error(err)
		return
	}
	if err := db.Create(&room).Error; err != nil {
		c.Error(err)
		return
	}
	c.String(200, "房間創建成功，請登入後開始使用")
}

func index(c *gin.Context) {
	room := getRoom(c)
	if !room.IsLogin {
		c.Redirect(303, "/login")
		return
	}
	var replies []Reply
	if err := db.Model(&room).Related(&replies).Order("created_at desc").Limit(20).Find(&replies).Error; err != nil {
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
	}
	log.Println(r)
	if err := db.Create(&r).Error; err != nil {
		log.Panic(err)
		c.String(500, err.Error())
		return
	}
	c.String(200, short(r.CreatedAt)+" - "+r.Content)
	return
}

func short(t time.Time) string {
	return t.Format("15:04")
}

func init() {
	var err error
	db, err = gorm.Open("sqlite3", "test.sqlite")
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
		if getRoom(c).IsLogin{
			c.Redirect(303, "/")
		}
		page := struct {
			Title  string
			Action string
		}{"註冊", "/create"}
		c.HTML(200, "login.gohtml", page)
	})
	r.POST("/create", create)
	r.Run()
}
