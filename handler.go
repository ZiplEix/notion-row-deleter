package main

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
)

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	running := atomic.LoadInt32(&deleting) == 1
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "index.html", struct {
		Running bool
		Error   string
	}{Running: running}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		token := r.FormValue("token")
		databaseID := r.FormValue("dbid")
		if token == "" || databaseID == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := templates.ExecuteTemplate(w, "index.html", struct {
				Running bool
				Error   string
			}{Running: false, Error: "Veuillez renseigner le token et l'ID de base."}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		if !atomic.CompareAndSwapInt32(&deleting, 0, 1) {
			http.Error(w, "Une suppression est déjà en cours.", http.StatusConflict)
			return
		}
		// Run in background to answer immediately.
		go func(tok, db string) {
			defer atomic.StoreInt32(&deleting, 0)
			if _, err := runDeletion(context.Background(), tok, db); err != nil {
				fmt.Println("Erreur suppression:", err)
			}
		}(token, databaseID)
		http.Redirect(w, r, "/deleting", http.StatusSeeOther)
	}
}

func deletingHandler(w http.ResponseWriter, r *http.Request) {
	running := atomic.LoadInt32(&deleting) == 1
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "deleting.html", struct{ Running bool }{Running: running}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func docsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "docs.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
