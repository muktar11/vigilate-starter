package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"vigilatestarter/internal/channeldata"
	"vigilatestarter/internal/config"
	"vigilatestarter/internal/driver"
	"vigilatestarter/internal/handlers"
	"vigilatestarter/internal/helpers"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/pusher/pusher-http-go"
	"github.com/robfig/cron/v3"
)

func setupApp() (*string, error) {
	// read flags
	insecurePort := flag.String("port", ":4000", "port to listen on")
	identifier := flag.String("identifier", "vigilate", "unique identifier")
	domain := flag.String("domain", "localhost", "domain name (e.g. example.com)")
	inProduction := flag.Bool("production", false, "application is in production")
	dbHost := flag.String("dbhost", "localhost", "database host")
	dbPort := flag.String("dbport", "5431", "database port")
	dbUser := flag.String("dbuser", "root", "database user")
	dbPass := flag.String("dbpass", "root", "database password")
	databaseName := flag.String("db", "vigilate", "database name")
	dbSsl := flag.String("dbssl", "disable", "database ssl setting")
	pusherHost := flag.String("pusherHost", "8080", "pusher host")
	pusherPort := flag.String("pusherPort", "4433", "pusher port")
	pusherApp := flag.String("pusherApp", "1", "pusher app id")
	pusherKey := flag.String("pusherKey", "307a37b4d49b930ede59", "pusher key")
	pusherSecret := flag.String("pusherSecret", "123abc", "pusher secret")
	pusherSecure := flag.Bool("pusherSecure", false, "pusher server uses SSL (true or false)")

	flag.Parse()

	if *dbUser == "" || *dbHost == "" || *dbPort == "" || *databaseName == "" || *identifier == "" {
		fmt.Println("Missing required flags.")
		os.Exit(1)
	}

	log.Println("Connecting to database....")
	dsnString := ""

	// when developing locally, we often don't have a db password
	if *dbPass == "" {
		dsnString = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s timezone=UTC connect_timeout=5",
			*dbHost,
			*dbPort,
			*dbUser,
			*databaseName,
			*dbSsl)
	} else {
		dsnString = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=UTC connect_timeout=5",
			*dbHost,
			*dbPort,
			*dbUser,
			*dbPass,
			*databaseName,
			*dbSsl)
	}

	db, err := driver.ConnectPostgres(dsnString)
	if err != nil {
		log.Fatal("Cannot connect to database!", err)
	}

	// session
	log.Printf("Initializing session manager....")
	session = scs.New()
	session.Store = postgresstore.New(db.SQL)
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.Name = fmt.Sprintf("gbsession_id_%s", *identifier)
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = *inProduction

	// start mail channel
	log.Println("Initializing mail channel and worker pool....")
	mailQueue := make(chan channeldata.MailJob, maxWorkerPoolSize)

	// Start the email dispatcher
	log.Println("Starting email dispatcher....")
	dispatcher := NewDispatcher(mailQueue, maxJobMaxWorkers)
	dispatcher.run()

	// define application configuration
	a := config.AppConfig{
		DB:           db,
		Session:      session,
		InProduction: *inProduction,
		Domain:       *domain,
		PusherSecret: *pusherSecret,
		MailQueue:    mailQueue,
		Version:      vigilateVersion,
		Identifier:   *identifier,
	}

	app = a

	repo = handlers.NewPostgresqlHandlers(db, &app)
	handlers.NewHandlers(repo, &app)

	log.Println("Getting preferences...")
	preferenceMap = make(map[string]string)
	preferences, err := repo.DB.AllPreferences()
	if err != nil {
		log.Fatal("Cannot read preferences:", err)
	}

	for _, pref := range preferences {
		preferenceMap[pref.Name] = string(pref.Preference)
	}

	preferenceMap["pusher-host"] = *pusherHost
	preferenceMap["pusher-port"] = *pusherPort
	preferenceMap["pusher-key"] = *pusherKey
	preferenceMap["identifier"] = *identifier
	preferenceMap["version"] = vigilateVersion

	app.PreferenceMap = preferenceMap

	// create pusher client
	wsClient := pusher.Client{
		AppID: "1512388",
		Key: "307a37b4d49b930ede59",
		Secret: "565d038ae54c4262d686",
		Cluster: "ap2",
		Secure:  true,
	}
	/*
	wsClient = pusher.Client{
		AppID:  *pusherApp,
		Secret: *pusherSecret,
		Key:    *pusherKey,
		Secure: *pusherSecure,
		Host:   fmt.Sprintf("%s:%s", *pusherHost, *pusherPort),
	}
	*/

	log.Println("Host", fmt.Sprintf("%s:%s: %s", *&pusherApp,  *pusherHost, *pusherPort))
	log.Println("Secure", *pusherSecure)

	app.WsClient = wsClient 
	monitorMap := make(map[int]cron.EntryID)
	app.MonitorMap = monitorMap 
	//set timezone
	localZone, _ := time.LoadLocation("Local")
	scheduler := cron.New(cron.WithLocation(localZone), cron.WithChain(
		cron.DelayIfStillRunning(cron.DefaultLogger), 
		cron.Recover(cron.DefaultLogger),
	))

	app.Scheduler = scheduler 

	go handlers.Repo.StartMonitoring() 

	if app.PreferenceMap["monitoring_live"] == "1"{
		app.Scheduler.Start()
	}
	helpers.NewHelpers(&app)

	return insecurePort, err
}

// createDirIfNotExist creates a directory if it does not exist
func createDirIfNotExist(path string) error {
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, mode)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}
