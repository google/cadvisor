// Copyright 2015 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package testing provides a fake implementation of the Docker API, useful for
// testing purpose.
package testing

import (
	"archive/tar"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	mathrand "math/rand"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/gorilla/mux"
)

var nameRegexp = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`)

// DockerServer represents a programmable, concurrent (not much), HTTP server
// implementing a fake version of the Docker remote API.
//
// It can used in standalone mode, listening for connections or as an arbitrary
// HTTP handler.
//
// For more details on the remote API, check http://goo.gl/G3plxW.
type DockerServer struct {
	containers     []*docker.Container
	execs          []*docker.ExecInspect
	execMut        sync.RWMutex
	cMut           sync.RWMutex
	images         []docker.Image
	iMut           sync.RWMutex
	imgIDs         map[string]string
	listener       net.Listener
	mux            *mux.Router
	hook           func(*http.Request)
	failures       map[string]string
	multiFailures  []map[string]string
	execCallbacks  map[string]func()
	customHandlers map[string]http.Handler
	handlerMutex   sync.RWMutex
	cChan          chan<- *docker.Container
}

// NewServer returns a new instance of the fake server, in standalone mode. Use
// the method URL to get the URL of the server.
//
// It receives the bind address (use 127.0.0.1:0 for getting an available port
// on the host), a channel of containers and a hook function, that will be
// called on every request.
//
// The fake server will send containers in the channel whenever the container
// changes its state, via the HTTP API (i.e.: create, start and stop). This
// channel may be nil, which means that the server won't notify on state
// changes.
func NewServer(bind string, containerChan chan<- *docker.Container, hook func(*http.Request)) (*DockerServer, error) {
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, err
	}
	server := DockerServer{
		listener:       listener,
		imgIDs:         make(map[string]string),
		hook:           hook,
		failures:       make(map[string]string),
		execCallbacks:  make(map[string]func()),
		customHandlers: make(map[string]http.Handler),
		cChan:          containerChan,
	}
	server.buildMuxer()
	go http.Serve(listener, &server)
	return &server, nil
}

func (s *DockerServer) notify(container *docker.Container) {
	if s.cChan != nil {
		s.cChan <- container
	}
}

func (s *DockerServer) buildMuxer() {
	s.mux = mux.NewRouter()
	s.mux.Path("/commit").Methods("POST").HandlerFunc(s.handlerWrapper(s.commitContainer))
	s.mux.Path("/containers/json").Methods("GET").HandlerFunc(s.handlerWrapper(s.listContainers))
	s.mux.Path("/containers/create").Methods("POST").HandlerFunc(s.handlerWrapper(s.createContainer))
	s.mux.Path("/containers/{id:.*}/json").Methods("GET").HandlerFunc(s.handlerWrapper(s.inspectContainer))
	s.mux.Path("/containers/{id:.*}/rename").Methods("POST").HandlerFunc(s.handlerWrapper(s.renameContainer))
	s.mux.Path("/containers/{id:.*}/top").Methods("GET").HandlerFunc(s.handlerWrapper(s.topContainer))
	s.mux.Path("/containers/{id:.*}/start").Methods("POST").HandlerFunc(s.handlerWrapper(s.startContainer))
	s.mux.Path("/containers/{id:.*}/kill").Methods("POST").HandlerFunc(s.handlerWrapper(s.stopContainer))
	s.mux.Path("/containers/{id:.*}/stop").Methods("POST").HandlerFunc(s.handlerWrapper(s.stopContainer))
	s.mux.Path("/containers/{id:.*}/pause").Methods("POST").HandlerFunc(s.handlerWrapper(s.pauseContainer))
	s.mux.Path("/containers/{id:.*}/unpause").Methods("POST").HandlerFunc(s.handlerWrapper(s.unpauseContainer))
	s.mux.Path("/containers/{id:.*}/wait").Methods("POST").HandlerFunc(s.handlerWrapper(s.waitContainer))
	s.mux.Path("/containers/{id:.*}/attach").Methods("POST").HandlerFunc(s.handlerWrapper(s.attachContainer))
	s.mux.Path("/containers/{id:.*}").Methods("DELETE").HandlerFunc(s.handlerWrapper(s.removeContainer))
	s.mux.Path("/containers/{id:.*}/exec").Methods("POST").HandlerFunc(s.handlerWrapper(s.createExecContainer))
	s.mux.Path("/exec/{id:.*}/resize").Methods("POST").HandlerFunc(s.handlerWrapper(s.resizeExecContainer))
	s.mux.Path("/exec/{id:.*}/start").Methods("POST").HandlerFunc(s.handlerWrapper(s.startExecContainer))
	s.mux.Path("/exec/{id:.*}/json").Methods("GET").HandlerFunc(s.handlerWrapper(s.inspectExecContainer))
	s.mux.Path("/images/create").Methods("POST").HandlerFunc(s.handlerWrapper(s.pullImage))
	s.mux.Path("/build").Methods("POST").HandlerFunc(s.handlerWrapper(s.buildImage))
	s.mux.Path("/images/json").Methods("GET").HandlerFunc(s.handlerWrapper(s.listImages))
	s.mux.Path("/images/{id:.*}").Methods("DELETE").HandlerFunc(s.handlerWrapper(s.removeImage))
	s.mux.Path("/images/{name:.*}/json").Methods("GET").HandlerFunc(s.handlerWrapper(s.inspectImage))
	s.mux.Path("/images/{name:.*}/push").Methods("POST").HandlerFunc(s.handlerWrapper(s.pushImage))
	s.mux.Path("/images/{name:.*}/tag").Methods("POST").HandlerFunc(s.handlerWrapper(s.tagImage))
	s.mux.Path("/events").Methods("GET").HandlerFunc(s.listEvents)
	s.mux.Path("/_ping").Methods("GET").HandlerFunc(s.handlerWrapper(s.pingDocker))
	s.mux.Path("/images/load").Methods("POST").HandlerFunc(s.handlerWrapper(s.loadImage))
	s.mux.Path("/images/{id:.*}/get").Methods("GET").HandlerFunc(s.handlerWrapper(s.getImage))
}

// SetHook changes the hook function used by the server.
//
// The hook function is a function called on every request.
func (s *DockerServer) SetHook(hook func(*http.Request)) {
	s.hook = hook
}

// PrepareExec adds a callback to a container exec in the fake server.
//
// This function will be called whenever the given exec id is started, and the
// given exec id will remain in the "Running" start while the function is
// running, so it's useful for emulating an exec that runs for two seconds, for
// example:
//
//    opts := docker.CreateExecOptions{
//        AttachStdin:  true,
//        AttachStdout: true,
//        AttachStderr: true,
//        Tty:          true,
//        Cmd:          []string{"/bin/bash", "-l"},
//    }
//    // Client points to a fake server.
//    exec, err := client.CreateExec(opts)
//    // handle error
//    server.PrepareExec(exec.ID, func() {time.Sleep(2 * time.Second)})
//    err = client.StartExec(exec.ID, docker.StartExecOptions{Tty: true}) // will block for 2 seconds
//    // handle error
func (s *DockerServer) PrepareExec(id string, callback func()) {
	s.execCallbacks[id] = callback
}

// PrepareFailure adds a new expected failure based on a URL regexp it receives
// an id for the failure.
func (s *DockerServer) PrepareFailure(id string, urlRegexp string) {
	s.failures[id] = urlRegexp
}

// PrepareMultiFailures enqueues a new expected failure based on a URL regexp
// it receives an id for the failure.
func (s *DockerServer) PrepareMultiFailures(id string, urlRegexp string) {
	s.multiFailures = append(s.multiFailures, map[string]string{"error": id, "url": urlRegexp})
}

// ResetFailure removes an expected failure identified by the given id.
func (s *DockerServer) ResetFailure(id string) {
	delete(s.failures, id)
}

// ResetMultiFailures removes all enqueued failures.
func (s *DockerServer) ResetMultiFailures() {
	s.multiFailures = []map[string]string{}
}

// CustomHandler registers a custom handler for a specific path.
//
// For example:
//
//     server.CustomHandler("/containers/json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//         http.Error(w, "Something wrong is not right", http.StatusInternalServerError)
//     }))
func (s *DockerServer) CustomHandler(path string, handler http.Handler) {
	s.handlerMutex.Lock()
	s.customHandlers[path] = handler
	s.handlerMutex.Unlock()
}

// MutateContainer changes the state of a container, returning an error if the
// given id does not match to any container "running" in the server.
func (s *DockerServer) MutateContainer(id string, state docker.State) error {
	for _, container := range s.containers {
		if container.ID == id {
			container.State = state
			return nil
		}
	}
	return errors.New("container not found")
}

// Stop stops the server.
func (s *DockerServer) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
}

// URL returns the HTTP URL of the server.
func (s *DockerServer) URL() string {
	if s.listener == nil {
		return ""
	}
	return "http://" + s.listener.Addr().String() + "/"
}

// ServeHTTP handles HTTP requests sent to the server.
func (s *DockerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handlerMutex.RLock()
	defer s.handlerMutex.RUnlock()
	for re, handler := range s.customHandlers {
		if m, _ := regexp.MatchString(re, r.URL.Path); m {
			handler.ServeHTTP(w, r)
			return
		}
	}
	s.mux.ServeHTTP(w, r)
	if s.hook != nil {
		s.hook(r)
	}
}

// DefaultHandler returns default http.Handler mux, it allows customHandlers to
// call the default behavior if wanted.
func (s *DockerServer) DefaultHandler() http.Handler {
	return s.mux
}

func (s *DockerServer) handlerWrapper(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		for errorID, urlRegexp := range s.failures {
			matched, err := regexp.MatchString(urlRegexp, r.URL.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if !matched {
				continue
			}
			http.Error(w, errorID, http.StatusBadRequest)
			return
		}
		for i, failure := range s.multiFailures {
			matched, err := regexp.MatchString(failure["url"], r.URL.Path)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if !matched {
				continue
			}
			http.Error(w, failure["error"], http.StatusBadRequest)
			s.multiFailures = append(s.multiFailures[:i], s.multiFailures[i+1:]...)
			return
		}
		f(w, r)
	}
}

func (s *DockerServer) listContainers(w http.ResponseWriter, r *http.Request) {
	all := r.URL.Query().Get("all")
	s.cMut.RLock()
	result := make([]docker.APIContainers, len(s.containers))
	for i, container := range s.containers {
		if all == "1" || container.State.Running {
			result[i] = docker.APIContainers{
				ID:      container.ID,
				Image:   container.Image,
				Command: fmt.Sprintf("%s %s", container.Path, strings.Join(container.Args, " ")),
				Created: container.Created.Unix(),
				Status:  container.State.String(),
				Ports:   container.NetworkSettings.PortMappingAPI(),
				Names:   []string{fmt.Sprintf("/%s", container.Name)},
			}
		}
	}
	s.cMut.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (s *DockerServer) listImages(w http.ResponseWriter, r *http.Request) {
	s.cMut.RLock()
	result := make([]docker.APIImages, len(s.images))
	for i, image := range s.images {
		result[i] = docker.APIImages{
			ID:      image.ID,
			Created: image.Created.Unix(),
		}
		for tag, id := range s.imgIDs {
			if id == image.ID {
				result[i].RepoTags = append(result[i].RepoTags, tag)
			}
		}
	}
	s.cMut.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (s *DockerServer) findImage(id string) (string, error) {
	s.iMut.RLock()
	defer s.iMut.RUnlock()
	image, ok := s.imgIDs[id]
	if ok {
		return image, nil
	}
	image, _, err := s.findImageByID(id)
	return image, err
}

func (s *DockerServer) findImageByID(id string) (string, int, error) {
	s.iMut.RLock()
	defer s.iMut.RUnlock()
	for i, image := range s.images {
		if image.ID == id {
			return image.ID, i, nil
		}
	}
	return "", -1, errors.New("No such image")
}

func (s *DockerServer) createContainer(w http.ResponseWriter, r *http.Request) {
	var config docker.Config
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := r.URL.Query().Get("name")
	if name != "" && !nameRegexp.MatchString(name) {
		http.Error(w, "Invalid container name", http.StatusInternalServerError)
		return
	}
	if _, err := s.findImage(config.Image); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusCreated)
	ports := map[docker.Port][]docker.PortBinding{}
	for port := range config.ExposedPorts {
		ports[port] = []docker.PortBinding{{
			HostIP:   "0.0.0.0",
			HostPort: strconv.Itoa(mathrand.Int() % 65536),
		}}
	}

	//the container may not have cmd when using a Dockerfile
	var path string
	var args []string
	if len(config.Cmd) == 1 {
		path = config.Cmd[0]
	} else if len(config.Cmd) > 1 {
		path = config.Cmd[0]
		args = config.Cmd[1:]
	}

	container := docker.Container{
		Name:    name,
		ID:      s.generateID(),
		Created: time.Now(),
		Path:    path,
		Args:    args,
		Config:  &config,
		State: docker.State{
			Running:   false,
			Pid:       mathrand.Int() % 50000,
			ExitCode:  0,
			StartedAt: time.Now(),
		},
		Image: config.Image,
		NetworkSettings: &docker.NetworkSettings{
			IPAddress:   fmt.Sprintf("172.16.42.%d", mathrand.Int()%250+2),
			IPPrefixLen: 24,
			Gateway:     "172.16.42.1",
			Bridge:      "docker0",
			Ports:       ports,
		},
	}
	s.cMut.Lock()
	s.containers = append(s.containers, &container)
	s.cMut.Unlock()
	s.notify(&container)
	var c = struct{ ID string }{ID: container.ID}
	json.NewEncoder(w).Encode(c)
}

func (s *DockerServer) generateID() string {
	var buf [16]byte
	rand.Read(buf[:])
	return fmt.Sprintf("%x", buf)
}

func (s *DockerServer) renameContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	container, index, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	copy := *container
	copy.Name = r.URL.Query().Get("name")
	s.cMut.Lock()
	defer s.cMut.Unlock()
	if s.containers[index].ID == copy.ID {
		s.containers[index] = &copy
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *DockerServer) inspectContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	container, _, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(container)
}

func (s *DockerServer) topContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	container, _, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if !container.State.Running {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Container %s is not running", id)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	result := docker.TopResult{
		Titles: []string{"UID", "PID", "PPID", "C", "STIME", "TTY", "TIME", "CMD"},
		Processes: [][]string{
			{"root", "7535", "7516", "0", "03:20", "?", "00:00:00", container.Path + " " + strings.Join(container.Args, " ")},
		},
	}
	json.NewEncoder(w).Encode(result)
}

func (s *DockerServer) startContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	container, _, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.cMut.Lock()
	defer s.cMut.Unlock()
	if container.State.Running {
		http.Error(w, "Container already running", http.StatusBadRequest)
		return
	}
	container.State.Running = true
	s.notify(container)
}

func (s *DockerServer) stopContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	container, _, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.cMut.Lock()
	defer s.cMut.Unlock()
	if !container.State.Running {
		http.Error(w, "Container not running", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	container.State.Running = false
	s.notify(container)
}

func (s *DockerServer) pauseContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	container, _, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.cMut.Lock()
	defer s.cMut.Unlock()
	if container.State.Paused {
		http.Error(w, "Container already paused", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	container.State.Paused = true
}

func (s *DockerServer) unpauseContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	container, _, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	s.cMut.Lock()
	defer s.cMut.Unlock()
	if !container.State.Paused {
		http.Error(w, "Container not paused", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	container.State.Paused = false
}

func (s *DockerServer) attachContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	container, _, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	outStream := newStdWriter(w, stdout)
	fmt.Fprintf(outStream, "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.docker.raw-stream\r\n\r\n")
	if container.State.Running {
		fmt.Fprintf(outStream, "Container %q is running\n", container.ID)
	} else {
		fmt.Fprintf(outStream, "Container %q is not running\n", container.ID)
	}
	fmt.Fprintln(outStream, "What happened?")
	fmt.Fprintln(outStream, "Something happened")
}

func (s *DockerServer) waitContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	container, _, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	for {
		time.Sleep(1e6)
		s.cMut.RLock()
		if !container.State.Running {
			s.cMut.RUnlock()
			break
		}
		s.cMut.RUnlock()
	}
	result := map[string]int{"StatusCode": container.State.ExitCode}
	json.NewEncoder(w).Encode(result)
}

func (s *DockerServer) removeContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	_, index, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if s.containers[index].State.Running {
		msg := "Error: API error (406): Impossible to remove a running container, please stop it first"
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	s.cMut.Lock()
	defer s.cMut.Unlock()
	s.containers[index] = s.containers[len(s.containers)-1]
	s.containers = s.containers[:len(s.containers)-1]
}

func (s *DockerServer) commitContainer(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("container")
	container, _, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var config *docker.Config
	runConfig := r.URL.Query().Get("run")
	if runConfig != "" {
		config = new(docker.Config)
		err = json.Unmarshal([]byte(runConfig), config)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	image := docker.Image{
		ID:        "img-" + container.ID,
		Parent:    container.Image,
		Container: container.ID,
		Comment:   r.URL.Query().Get("m"),
		Author:    r.URL.Query().Get("author"),
		Config:    config,
	}
	repository := r.URL.Query().Get("repo")
	tag := r.URL.Query().Get("tag")
	s.iMut.Lock()
	s.images = append(s.images, image)
	if repository != "" {
		if tag != "" {
			repository += ":" + tag
		}
		s.imgIDs[repository] = image.ID
	}
	s.iMut.Unlock()
	fmt.Fprintf(w, `{"ID":%q}`, image.ID)
}

func (s *DockerServer) findContainer(id string) (*docker.Container, int, error) {
	s.cMut.RLock()
	defer s.cMut.RUnlock()
	for i, container := range s.containers {
		if container.ID == id {
			return container, i, nil
		}
	}
	return nil, -1, errors.New("No such container")
}

func (s *DockerServer) buildImage(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); ct == "application/tar" {
		gotDockerFile := false
		tr := tar.NewReader(r.Body)
		for {
			header, err := tr.Next()
			if err != nil {
				break
			}
			if header.Name == "Dockerfile" {
				gotDockerFile = true
			}
		}
		if !gotDockerFile {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("miss Dockerfile"))
			return
		}
	}
	//we did not use that Dockerfile to build image cause we are a fake Docker daemon
	image := docker.Image{
		ID:      s.generateID(),
		Created: time.Now(),
	}

	query := r.URL.Query()
	repository := image.ID
	if t := query.Get("t"); t != "" {
		repository = t
	}
	s.iMut.Lock()
	s.images = append(s.images, image)
	s.imgIDs[repository] = image.ID
	s.iMut.Unlock()
	w.Write([]byte(fmt.Sprintf("Successfully built %s", image.ID)))
}

func (s *DockerServer) pullImage(w http.ResponseWriter, r *http.Request) {
	fromImageName := r.URL.Query().Get("fromImage")
	tag := r.URL.Query().Get("tag")
	image := docker.Image{
		ID: s.generateID(),
	}
	s.iMut.Lock()
	s.images = append(s.images, image)
	if fromImageName != "" {
		if tag != "" {
			fromImageName = fmt.Sprintf("%s:%s", fromImageName, tag)
		}
		s.imgIDs[fromImageName] = image.ID
	}
	s.iMut.Unlock()
}

func (s *DockerServer) pushImage(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	tag := r.URL.Query().Get("tag")
	if tag != "" {
		name += ":" + tag
	}
	s.iMut.RLock()
	if _, ok := s.imgIDs[name]; !ok {
		s.iMut.RUnlock()
		http.Error(w, "No such image", http.StatusNotFound)
		return
	}
	s.iMut.RUnlock()
	fmt.Fprintln(w, "Pushing...")
	fmt.Fprintln(w, "Pushed")
}

func (s *DockerServer) tagImage(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	s.iMut.RLock()
	if _, ok := s.imgIDs[name]; !ok {
		s.iMut.RUnlock()
		http.Error(w, "No such image", http.StatusNotFound)
		return
	}
	s.iMut.RUnlock()
	s.iMut.Lock()
	defer s.iMut.Unlock()
	newRepo := r.URL.Query().Get("repo")
	newTag := r.URL.Query().Get("tag")
	if newTag != "" {
		newRepo += ":" + newTag
	}
	s.imgIDs[newRepo] = s.imgIDs[name]
	w.WriteHeader(http.StatusCreated)
}

func (s *DockerServer) removeImage(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	s.iMut.RLock()
	var tag string
	if img, ok := s.imgIDs[id]; ok {
		id, tag = img, id
	}
	var tags []string
	for tag, taggedID := range s.imgIDs {
		if taggedID == id {
			tags = append(tags, tag)
		}
	}
	s.iMut.RUnlock()
	_, index, err := s.findImageByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	s.iMut.Lock()
	defer s.iMut.Unlock()
	if len(tags) < 2 {
		s.images[index] = s.images[len(s.images)-1]
		s.images = s.images[:len(s.images)-1]
	}
	if tag != "" {
		delete(s.imgIDs, tag)
	}
}

func (s *DockerServer) inspectImage(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if id, ok := s.imgIDs[name]; ok {
		s.iMut.Lock()
		defer s.iMut.Unlock()

		for _, img := range s.images {
			if img.ID == id {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(img)
				return
			}
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func (s *DockerServer) listEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var events [][]byte
	count := mathrand.Intn(20)
	for i := 0; i < count; i++ {
		data, err := json.Marshal(s.generateEvent())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		events = append(events, data)
	}
	w.WriteHeader(http.StatusOK)
	for _, d := range events {
		fmt.Fprintln(w, d)
		time.Sleep(time.Duration(mathrand.Intn(200)) * time.Millisecond)
	}
}

func (s *DockerServer) pingDocker(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *DockerServer) generateEvent() *docker.APIEvents {
	var eventType string
	switch mathrand.Intn(4) {
	case 0:
		eventType = "create"
	case 1:
		eventType = "start"
	case 2:
		eventType = "stop"
	case 3:
		eventType = "destroy"
	}
	return &docker.APIEvents{
		ID:     s.generateID(),
		Status: eventType,
		From:   "mybase:latest",
		Time:   time.Now().Unix(),
	}
}

func (s *DockerServer) loadImage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *DockerServer) getImage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/tar")
}

func (s *DockerServer) createExecContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	container, _, err := s.findContainer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	exec := docker.ExecInspect{
		ID:        s.generateID(),
		Container: *container,
	}
	var params docker.CreateExecOptions
	err = json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(params.Cmd) > 0 {
		exec.ProcessConfig.EntryPoint = params.Cmd[0]
		if len(params.Cmd) > 1 {
			exec.ProcessConfig.Arguments = params.Cmd[1:]
		}
	}
	s.execMut.Lock()
	s.execs = append(s.execs, &exec)
	s.execMut.Unlock()
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"Id": exec.ID})
}

func (s *DockerServer) startExecContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if exec, err := s.getExec(id); err == nil {
		s.execMut.Lock()
		exec.Running = true
		s.execMut.Unlock()
		if callback, ok := s.execCallbacks[id]; ok {
			callback()
			delete(s.execCallbacks, id)
		} else if callback, ok := s.execCallbacks["*"]; ok {
			callback()
			delete(s.execCallbacks, "*")
		}
		s.execMut.Lock()
		exec.Running = false
		s.execMut.Unlock()
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *DockerServer) resizeExecContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if _, err := s.getExec(id); err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *DockerServer) inspectExecContainer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if exec, err := s.getExec(id); err == nil {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(exec)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *DockerServer) getExec(id string) (*docker.ExecInspect, error) {
	s.execMut.RLock()
	defer s.execMut.RUnlock()
	for _, exec := range s.execs {
		if exec.ID == id {
			return exec, nil
		}
	}
	return nil, errors.New("exec not found")
}
