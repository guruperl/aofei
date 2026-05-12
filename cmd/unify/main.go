// go run summer.go --log_dir="../../logs/"
// this is the web server for the summer and aofei projects
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/genelet/winter/dsp"
	"github.com/genelet/winter/genelet"
	"github.com/genelet/winter/summer/registry"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: summer --g=web_config --s=ssp_config\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var gConf, sConf string
var isLocal bool

func init() {
	flag.Usage = usage
	flag.StringVar(&gConf, "g", os.Getenv("SUMMER"), "Genelet Config")
	flag.StringVar(&sConf, "s", os.Getenv("AOFEI"), "Ssp Config")
	flag.BoolVar(&isLocal, "local", false, "local mode")
	flag.Parse()
}

func main() {
	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	sc, err := dsp.NewController(ctx, sConf)
	if err != nil {
		log.Fatal(err)
	}
	sc.C.IsLocal = isLocal
	sc.Logger = logger
	defer sc.Close()

	gc, err := getGenelet(gConf, logger)
	if err != nil {
		log.Fatal(err)
	}
	gc.DB = sc.DB
	gc.Storage["Redis"] = sc.Redis
	gc.Storage["Nc"] = sc.Nc
	gc.Storage["Spread"] = sc.C.Spread // pass the top to genelet, so it can use the same IO for local mode
	if gc.C.ServerPort == "" {
		gc.C.ServerPort = sc.C.ServerPort
	}
	if gc.C.ServerURL == "" {
		gc.C.ServerURL = sc.C.ServerURL
	}
	if gc.C.DocumentRoot == "" {
		gc.C.DocumentRoot = sc.C.DocumentRoot
	}
	if gc.C.ConnectArray == nil {
		gc.C.ConnectArray = sc.C.ConnectArray
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /bid/{domain}", sc.ServeBid)
	mux.HandleFunc("GET /win", sc.ServeWinLoss)
	mux.HandleFunc("GET /loss", sc.ServeWinLoss)
	mux.HandleFunc("GET /clk", sc.ServeWinLoss)
	mux.HandleFunc("GET /imp", sc.ServeWinLoss)
	mux.Handle("/", gc)

	server := &http.Server{
		Addr:           ":" + sc.C.ServerPort,
		Handler:        mux,
		ReadTimeout:    15 * time.Second, // 15 seconds
		WriteTimeout:   15 * time.Second, // 15 seconds
		MaxHeaderBytes: 1 << 20,          // 1 MB
	}

	// This is a blocking call, so it will not return until the server is stopped
	// or an error occurs.
	err = server.ListenAndServe()
	if err != nil && err == http.ErrServerClosed {
		log.Println("Server closed gracefully")
	} else if err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func getGenelet(fn string, logger *zap.Logger) (*genelet.Controller, error) {
	c, err := genelet.NewConfig(fn)
	if err != nil {
		return nil, err
	}
	models, storage, filters := registry.Build()
	for _, entry := range registry.Entries {
		comp, err := genelet.LoadComponent(c.ProjectRoot + "/summer/" + entry.Name + "/component.json")
		if err != nil {
			return nil, err
		}
		if err := genelet.InvokeVoid(models[entry.Name], "Initialize", comp, logger); err != nil {
			return nil, err
		}
		if err := genelet.InvokeVoid(storage[entry.Name], "Initialize", comp, logger); err != nil {
			return nil, err
		}
		if err := genelet.InvokeVoid(filters[entry.Name], "Initialize", comp, logger); err != nil {
			return nil, err
		}
	}

	return &genelet.Controller{
		C:       c,
		Models:  models,
		Filters: filters,
		Storage: storage,
		Logger:  logger,
	}, nil
}
