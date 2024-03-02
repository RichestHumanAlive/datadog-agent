// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

type apiResponse struct {
	Error *apiError `json:"error,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
}

// LocalAPI is the interface for the locally exposed API to interact with the updater.
type LocalAPI interface {
	Start(context.Context) error
	Stop(context.Context) error
}

// localAPIImpl is a locally exposed API to interact with the updater.
type localAPIImpl struct {
	updater  Updater
	listener net.Listener
	server   *http.Server
}

// NewLocalAPI returns a new LocalAPI.
func NewLocalAPI(updater Updater) (LocalAPI, error) {
	socketPath := path.Join(defaultRepositoriesPath, "updater.sock")
	err := os.RemoveAll(socketPath)
	if err != nil {
		return nil, fmt.Errorf("could not remove socket: %w", err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(socketPath, 0700); err != nil {
		return nil, fmt.Errorf("error setting socket permissions: %v", err)
	}
	return &localAPIImpl{
		server:   &http.Server{},
		listener: listener,
		updater:  updater,
	}, nil
}

// Start starts the LocalAPI.
func (l *localAPIImpl) Start(_ context.Context) error {
	l.server.Handler = l.handler()
	go func() {
		err := l.server.Serve(l.listener)
		if err != nil {
			log.Infof("Local API server stopped: %v", err)
		}
	}()
	return nil
}

// Stop stops the LocalAPI.
func (l *localAPIImpl) Stop(ctx context.Context) error {
	return l.server.Shutdown(ctx)
}

func (l *localAPIImpl) handler() http.Handler {
	r := mux.NewRouter().Headers("Content-Type", "application/json").Subrouter()
	r.HandleFunc("/{package}/experiment/start", l.startExperiment).Methods(http.MethodPost)
	r.HandleFunc("/{package}/experiment/stop", l.stopExperiment).Methods(http.MethodPost)
	r.HandleFunc("/{package}/experiment/promote", l.promoteExperiment).Methods(http.MethodPost)
	return r
}

// example: curl -X POST --unix-socket /opt/datadog-packages/updater.sock -H 'Content-Type: application/json' http://updater/datadog-agent/experiment/start -d '{"version":"1.21.5"}'
func (l *localAPIImpl) startExperiment(w http.ResponseWriter, r *http.Request) {
	pkg := mux.Vars(r)["package"]
	w.Header().Set("Content-Type", "application/json")
	var request startExperimentParams
	var response apiResponse
	defer func() {
		_ = json.NewEncoder(w).Encode(response)
	}()
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response.Error = &apiError{Message: err.Error()}
		return
	}
	log.Infof("Received local request to start experiment for package %s version %s", pkg, request.Version)
	err = l.updater.StartExperiment(r.Context(), pkg, request.Version)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Error = &apiError{Message: err.Error()}
		return
	}
}

// example: curl -X POST --unix-socket /opt/datadog-packages/updater.sock -H 'Content-Type: application/json' http://updater/datadog-agent/experiment/stop -d '{}'
func (l *localAPIImpl) stopExperiment(w http.ResponseWriter, r *http.Request) {
	pkg := mux.Vars(r)["package"]
	w.Header().Set("Content-Type", "application/json")
	var response apiResponse
	defer func() {
		_ = json.NewEncoder(w).Encode(response)
	}()
	log.Infof("Received local request to stop experiment for package %s", pkg)
	err := l.updater.StopExperiment(pkg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Error = &apiError{Message: err.Error()}
		return
	}
}

// example: curl -X POST --unix-socket /opt/datadog-packages/updater.sock -H 'Content-Type: application/json' http://updater/datadog-agent/experiment/promote -d '{}'
func (l *localAPIImpl) promoteExperiment(w http.ResponseWriter, r *http.Request) {
	pkg := mux.Vars(r)["package"]
	w.Header().Set("Content-Type", "application/json")
	var response apiResponse
	defer func() {
		_ = json.NewEncoder(w).Encode(response)
	}()
	log.Infof("Received local request to promote experiment for package %s", pkg)
	err := l.updater.PromoteExperiment(pkg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.Error = &apiError{Message: err.Error()}
		return
	}
}
