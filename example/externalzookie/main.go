package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
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
	Key        *datastore.Key `datastore:"__key__" json:"key"`
	Name       string         `datastore:"name" json:"name"`
	Version    int64          `datastore:"-" json:"version"`
	CreateTime time.Time      `datastore:"-" json:"create_time"`
	UpdateTime time.Time      `datastore:"-" json:"update_time"`
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
	http.HandleFunc("/task", h.taskHandler)
	http.ListenAndServe(":8080", http.DefaultServeMux)
}

func (h *exampleApp) taskHandler(w http.ResponseWriter, r *http.Request) {
	ctx, rec := datastoreextensions.WithRecorder(r.Context())
	var task Task

	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp, err := h.dex.Mutate(ctx, []*datastore.Mutation{datastore.NewInsert(datastore.IncompleteKey("Task", nil), &task)})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		task.Key, _ = datastoreextensions.ProtoToKey(resp.MutationResults[0].Key)
		task.Version = resp.MutationResults[0].Version
		task.CreateTime = resp.MutationResults[0].CreateTime.AsTime()
		task.UpdateTime = resp.MutationResults[0].UpdateTime.AsTime()
		_ = json.NewEncoder(w).Encode(task)
		return
	}

	// identify the task by its key
	key, err := datastore.DecodeKey(r.URL.Query().Get("key"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodDelete {
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mut := datastore.NewDelete(key)
		datastoreextensions.SetBaseVersion(mut, task.Version)
		resp, err := h.dex.Mutate(ctx, []*datastore.Mutation{mut})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": resp.MutationResults[0].Version})
		return
	}

	if r.Method == http.MethodGet {
		err = h.client.Get(ctx, key, &task)
		if errors.Is(err, datastore.ErrNoSuchEntity) {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if res, ok := rec.Get(key); ok {
			// expose datastore version and datastore timestamps
			task.Version = res.Version
			task.CreateTime = res.CreateTime.AsTime()
			task.UpdateTime = res.UpdateTime.AsTime()
		}
		_ = json.NewEncoder(w).Encode(task)
		return
	}

	if r.Method == http.MethodPut {
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mut := datastore.NewUpdate(key, &task)
		datastoreextensions.SetBaseVersion(mut, task.Version)
		resp, err := h.dex.Mutate(ctx, []*datastore.Mutation{mut})
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"version": resp.MutationResults[0].Version})
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
