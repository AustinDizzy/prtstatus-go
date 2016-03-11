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
}

func main() {
	logpkg.Println("Starting server...")
	duration, _ = time.ParseDuration(viper.GetString("refreshInterval"))

	var (
		ticker = time.NewTicker(duration)
		quit   = make(chan struct{})
		m      = macaron.Classic()
	)

	initDatabase(viper.GetString("postgres.user"), viper.GetString("postgres.db"))

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

	m.Post("/user", APIMiddleware(userHandler))
	m.Get("/api/data", APIMiddleware(dataAPI))
	m.Use(macaron.Static("static"))
	// router.HandleFunc("/auth", AuthHandler).Methods("GET")
	// router.HandleFunc("/store", CallbackHandler)
	// router.HandleFunc("/api", ApiRoot)
	// router.HandleFunc("/api/", ApiRoot)
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
