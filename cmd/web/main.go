// Package main is a placeholder for the future web UI.
// It imports the same todo.Service used by the desktop app,
// so the business logic and SQLite store are shared without duplication.
//
// To implement: add an HTTP router (e.g. chi or net/http ServeMux),
// register handlers that call svc.Add / svc.All / svc.Toggle / svc.Delete,
// and serve a REST API or server-rendered HTML frontend.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/user/todoapp/internal/store"
	"github.com/user/todoapp/internal/todo"
)

func main() {
	home, _ := os.UserHomeDir()
	st, err := store.New(filepath.Join(home, ".todoapp", "tasks.db"))
	if err != nil {
		log.Fatal(err)
	}
	svc := todo.NewService(st)
	defer svc.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tasks, _ := svc.All()
		fmt.Fprintf(w, "TodoApp web UI — %d tasks (implement me!)\n", len(tasks))
	})

	addr := ":8080"
	log.Printf("Web UI listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
