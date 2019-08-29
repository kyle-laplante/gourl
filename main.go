package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type Model struct {
	ID        uint      `gorm:"primary_key" json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type ShortUrl struct {
	Model
	ShortName string `json:"short_name"`
	Url       string `gorm:"index:url" json:"url"`
	ParamUrl  string `json:"param_url"`
}

func setupRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		var shortUrls []ShortUrl
		if err := db.Find(&shortUrls).Error; err != nil {
			fmt.Println("Unable to find all short urls")
		}
		c.HTML(http.StatusOK, "index.tmpl", gin.H{"shortUrls": shortUrls})
	})

	r.GET("/:shortname", func(c *gin.Context) {
		shortname := c.Params.ByName("shortname")
		var shortUrl ShortUrl
		if err := db.Where(&ShortUrl{ShortName: shortname}).First(&shortUrl).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			redirectTo := shortUrl.Url
			switch c.ContentType() {
			case "application/json":
				c.JSON(http.StatusOK, gin.H{"redirectTo": redirectTo})
			default:
				c.Redirect(http.StatusFound, redirectTo)
			}
		}
	})

	r.GET("/:shortname/:param", func(c *gin.Context) {
		shortname := c.Params.ByName("shortname")
		param := c.Params.ByName("param")
		var shortUrl ShortUrl
		if err := db.Where(&ShortUrl{ShortName: shortname}).First(&shortUrl).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if shortUrl.ParamUrl == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("%s does not have a param url defined", shortname)})
		} else {
			redirectTo := fmt.Sprintf(shortUrl.ParamUrl, param)
			switch c.ContentType() {
			case "application/json":
				c.JSON(http.StatusOK, gin.H{"redirectTo": redirectTo})
			default:
				c.Redirect(http.StatusFound, redirectTo)
			}
		}
	})

	r.POST("/new", func(c *gin.Context) {
		shortname := c.PostForm("shortname")
		var shortUrl ShortUrl
		if db.Where(&ShortUrl{ShortName: shortname}).First(&shortUrl).RecordNotFound() == false {
			c.JSON(http.StatusBadRequest, gin.H{"error": "this shortname already exists"})
		} else {
			url := c.PostForm("url")
			paramUrl := c.PostForm("param_url")
			fmt.Println(fmt.Sprintf("Creating new shorturl %s=%s and %s", shortname, url, paramUrl))
			shortUrl := ShortUrl{ShortName: shortname, Url: url, ParamUrl: paramUrl}
			if err := db.Create(&shortUrl).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			} else {
				c.Redirect(http.StatusFound, "/")
			}
		}
	})

	return r
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(&ShortUrl{}).Error
}

func main() {
	db, err := gorm.Open("mysql", "gourl:gourl@tcp(localhost:3306)/gourl?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic(fmt.Sprintf("Unable to connect to db: %s", err.Error()))
	}
	defer db.Close()
	if err = migrate(db); err != nil {
		panic(fmt.Sprintf("Unable to run migration: %s", err.Error()))
	}
	r := setupRouter(db)
	// Listen and Server in 0.0.0.0:8080
	r.Run(":80")
}
