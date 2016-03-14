package main

import (
	"github.com/spf13/viper"
	"gopkg.in/macaron.v1"
	"gopkg.in/pg.v4"
	logpkg "log"
	"runtime"
	"time"
)

var (
	DB       *pg.DB
	duration time.Duration
)

func init() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetDefault("refreshInterval", "15s")
	viper.SetDefault("port", ":8000")
	log(viper.ReadInConfig())

	initDatabase(viper.GetString("postgres.user"), viper.GetString("postgres.db"))
}

func main() {
	logpkg.Println("Starting server...")
	var (
		duration, _ = time.ParseDuration(viper.GetString("refreshInterval"))
		ticker      = time.NewTicker(duration)
		quit        = make(chan struct{})
		m           = macaron.Classic()
	)

	go func() {
		for {
			select {
			case <-ticker.C:
				go poll()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	m.Use(macaron.Renderer(macaron.RenderOptions{
		Directory: "views",
		Layout:    "layout",
	}))
	m.Post("/user", APIMiddleware(userHandler))
	m.Get("/api/data", APIMiddleware(dataAPI))
	m.Get("/", indexHandler)
	m.Get("/data", dataHandler)
	m.Combo("/pb_auth").Get(pbUserAuth).Post(pbUserAuth)
	m.Use(macaron.Static("static"))
	m.Run(viper.GetInt("port"))
}

func log(err error, args ...interface{}) {
	pc := make([]uintptr, 10)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])

	if viper.GetBool("debug") {

		if err != nil {
			panic(err)
		}

		logpkg.Printf("RUNNING %s", f.Name())

		if len(args) > 0 {
			for i := range args {
				logpkg.Printf("%#v", args[i])
			}
		}
	}
	if err != nil {
		logpkg.Println(err)
	}
}
