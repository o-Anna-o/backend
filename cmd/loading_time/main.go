package main

// go run cmd/loading_time/main.go

import (
	"fmt"
	"net/http"

	"loading_time/internal/app/config"
	"loading_time/internal/app/dsn"
	"loading_time/internal/app/handler"
	"loading_time/internal/app/pkg"
	"loading_time/internal/app/repository"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	gin.SetMode(gin.ReleaseMode) // Убрать предупреждения debug

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		logrus.Infof("Incoming request: %s %s", c.Request.Method, c.Request.URL.Path)
	})

	// Загрузка HTML-шаблонов
	router.LoadHTMLGlob("templates/*.html")

	conf, err := config.NewConfig()
	if err != nil {
		logrus.Fatalf("error loading config: %v", err)
	}

	postgresString := dsn.FromEnv()
	fmt.Println(postgresString)

	rep, errRep := repository.New(postgresString, conf.RedisHost, conf.RedisPort)
	if errRep != nil {
		logrus.Fatalf("error initializing repository: %v", errRep)
	}

	http.DefaultClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
	}

	hand := handler.NewHandler(rep)

	// Middleware для _method
	router.Use(func(c *gin.Context) {
		if m := c.PostForm("_method"); m != "" {
			logrus.Infof("Overriding method to %s for %s", m, c.Request.URL.Path)
			c.Request.Method = m
		}
		c.Next()
	})

	application := pkg.NewApp(conf, router, hand)
	application.RunApp()
}
