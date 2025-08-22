package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"time"

	"github.com/pkg/browser"
)

const notionVersion = "2022-06-28"

//go:embed templates/*.html public/*
var embeddedFS embed.FS

var templates = template.Must(template.ParseFS(embeddedFS, "templates/*.html"))

var deleting int32 // 0=idle, 1=running
var hub = newHub()

func main() {
	// Serve embedded static assets under /public/
	if pub, err := fs.Sub(embeddedFS, "public"); err == nil {
		http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.FS(pub))))
	} else {
		fmt.Println("Warning: public assets not available:", err)
	}
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/delete", deleteHandler())
	http.HandleFunc("/deleting", deletingHandler)
	http.HandleFunc("/docs", docsHandler)
	http.HandleFunc("/ws", wsHandler(hub))
	go hub.Run()

	addr := ":8080"
	fmt.Println("HTTP Server launched on", addr)
	// Try to open the default browser shortly after start
	go func() {
		time.Sleep(300 * time.Millisecond)
		err := browser.OpenURL("http://localhost:8080/")
		if err != nil {
			fmt.Println("Error opening browser:", err)
		}
	}()
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Println("HTTP server error:", err)
		os.Exit(1)
	}
}
