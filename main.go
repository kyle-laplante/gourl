package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

type Model struct {
	ID        uint      `gorm:"primary_key" json:"-"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type ShortUrl struct {
	Model
	ShortName string `gorm:"unique;not null" json:"short_name"`
	Url       string `gorm:"index:url;not null" json:"url"`
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

	r.POST("/", func(c *gin.Context) {
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

	r.DELETE("/:shortname", func(c *gin.Context) {
		shortname := c.Params.ByName("shortname")
		var shortUrl ShortUrl
		if err := db.Where(&ShortUrl{ShortName: shortname}).First(&shortUrl).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			if err := db.Unscoped().Delete(&shortUrl).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusOK, gin.H{"deleted": shortname})
			}
		}
	})

	return r
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(&ShortUrl{}).Error
}

func main() {
	port := flag.Int("port", 80, "the http port to serve on")
	dbHost := flag.String("dbHost", "localhost", "The database hostname")
	dbUser := flag.String("dbUser", "gourl", "The database user name")
	dbPass := flag.String("dbPass", "gourl", "The password for the database")
	flag.Parse()
	cxnString := fmt.Sprintf("host=%s port=5432 user=%s dbname=gourl password=%s sslmode=disable",
		*dbHost, *dbUser, *dbPass)
	db, err := gorm.Open("postgres", cxnString)
	if err != nil {
		panic(fmt.Sprintf("Unable to connect to db: %s", err.Error()))
	}
	defer db.Close()
	if err = migrate(db); err != nil {
		panic(fmt.Sprintf("Unable to run migration: %s", err.Error()))
	}
	r := setupRouter(db)
	// Listen and Server in 0.0.0.0:8080
	r.Run(fmt.Sprintf(":%d", *port))
}
