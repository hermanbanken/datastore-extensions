package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	datastoreextensions "github.com/hermanbanken/datastore-extensions"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

type exampleApp struct {
	client *datastore.Client
	dex    *datastoreextensions.DatastoreExtensionClient
}

type Task struct {
	Name  string         `datastore:"name"`
	Group *datastore.Key `datastore:"group"`
}

type TaskGroup struct {
	Name string `datastore:"name"`
}

func main() {
	projectID := os.Getenv("DATASTORE_PROJECT_ID")
	client, err := datastore.NewClient(context.Background(), projectID, option.WithGRPCDialOption(grpc.WithUnaryInterceptor(datastoreextensions.NewRecordingInterceptor())))
	if err != nil {
		panic(err)
	}
	dex, err := datastoreextensions.FromClient(client)
	if err != nil {
		panic(err)
	}

	h := &exampleApp{client, dex}
	http.HandleFunc("/group", h.groupHandler)
	http.HandleFunc("/task", h.taskHandler)
	http.ListenAndServe(":8080", http.DefaultServeMux)
}

// groupHandler inserts new groups
func (h *exampleApp) groupHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	keys, err := h.client.Mutate(r.Context(), datastore.NewInsert(datastore.IncompleteKey("TaskGroup", nil), &TaskGroup{Name: r.FormValue("name")}))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Group created with ID " + strconv.FormatInt(keys[0].ID, 10)))
}

// taskHandler depending on the method:
// - creates a new task with a group ID
// - reads the task and its group
// - updates the task
func (h *exampleApp) taskHandler(w http.ResponseWriter, r *http.Request) {
	ctx, _ := datastoreextensions.WithRecorder(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// insert
	if r.Method == http.MethodPost {
		group, err := strconv.ParseInt(r.FormValue("group"), 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		keys, err := h.client.Mutate(r.Context(), datastore.NewInsert(datastore.IncompleteKey("Task", nil), &Task{Group: datastore.IDKey("TaskGroup", group, nil)}))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte("Task created with ID " + strconv.FormatInt(keys[0].ID, 10)))
		return
	}

	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	var task Task
	var grp TaskGroup
	err = h.client.Get(ctx, datastore.IDKey("Task", id, nil), &task)
	if errors.Is(err, datastore.ErrNoSuchEntity) {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.client.Get(ctx, task.Group, &grp)
	log.Println(ctx.Value(datastoreextensions.RecorderKey))
	if errors.Is(err, datastore.ErrNoSuchEntity) {
		http.Error(w, "task group not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// lookup
	if r.Method == http.MethodGet {
		w.Write([]byte("Task: " + task.Name + " in group " + grp.Name))
		return
	}

	// update
	if r.Form.Has("demonstrateConflict") {
		// modify the group AFTER it is read to introduce a conflict
		_, err = h.client.Mutate(context.Background(), datastore.NewUpdate(task.Group, &TaskGroup{Name: strconv.Itoa(int(time.Now().Unix()))}))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	task.Name = r.FormValue("name")
	_, err = h.dex.MutateWithLocks(ctx, []*datastore.Mutation{datastore.NewUpdate(datastore.IDKey("Task", id, nil), &task)}, []*datastore.Key{task.Group})
	if err != nil {
		http.Error(w, fmt.Errorf("mutateWithLocks: %w", err).Error(), http.StatusInternalServerError)
		return
	}
}
