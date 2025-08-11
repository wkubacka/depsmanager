package main

import (
	"depsmanager"
	"depsmanager/clients"
	"depsmanager/service"
	"depsmanager/storage"
	"errors"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"log"
	"net/http"
	"time"
)

func main() {
	log.Println("Starting service")

	var conf depsmanager.Config
	envconfig.MustProcess("", &conf)

	db, err := storage.NewStorage(conf.SQLLiteConfig)
	if err != nil {
		log.Fatalf("storage.NewStorage: %s", err)
	}
	defer db.Close()
	log.Println("Storage started")

	svg := service.NewService(
		service.WithStorage(db),
		service.WithDepsClient(clients.NewDepsClient(conf.DepsAddress)),
		service.WithTimeNow(time.Now),
	)
	api := service.NewAPI(svg)

	log.Printf("Starting http server on port %d", conf.HTTPPort)
	httpServer := http.Server{Addr: fmt.Sprintf(":%d", conf.HTTPPort), Handler: api.GetHandler()}

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("httpServer.ListenAndServe: %s", err)
	}
}
