package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
)

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

func logout(c *gin.Context) {
	room := getRoom(c)
	if !room.IsLogin {
		c.Redirect(303, "/login")
		return
	}
	sessionID, err := c.Cookie("session")
	if err != nil {
		c.Redirect(303, "/login")
		return
	}
	if err := db.Where("id = ?", sessionID).Delete(&Session{}).Error; err != nil {
		log.Fatal(err)
	}
	c.Redirect(303, "/login")
}
