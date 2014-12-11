package pages

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/golang/glog"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/manager"
        auth "github.com/abbot/go-http-auth"
)

var pageTemplate *template.Template

type link struct {
	// Text to show in the link.
	Text string

	// Web address to link to.
	Link string
}

type pageData struct {
	DisplayName        string
	ContainerName      string
	ParentContainers   []link
	Subcontainers      []link
	Spec               info.ContainerSpec
	Stats              []*info.ContainerStats
	MachineInfo        *info.MachineInfo
	IsRoot             bool
	ResourcesAvailable bool
	CpuAvailable       bool
	MemoryAvailable    bool
	NetworkAvailable   bool
	FsAvailable        bool
}

func init() {
	pageTemplate = template.New("containersTemplate").Funcs(funcMap)
	_, err := pageTemplate.Parse(containersHtmlTemplate)
	if err != nil {
		glog.Fatalf("Failed to parse template: %s", err)
	}
}

func containerHandlerNoAuth(containerManager manager.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
        	err := serveContainersPage(containerManager, w, r.URL)
	        if err != nil {
	                fmt.Fprintf(w, "%s", err)
	        }
	}
}

func containerHandler(containerManager manager.Manager) auth.AuthenticatedHandlerFunc {
        return func(w http.ResponseWriter, r *auth.AuthenticatedRequest) { 
                err := serveContainersPage(containerManager, w, r.URL)
                if err != nil {
                        fmt.Fprintf(w, "%s", err)
                } 
        }
}

func dockerHandlerNoAuth(containerManager manager.Manager) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                err := serveDockerPage(containerManager, w, r.URL)
                if err != nil {
                        fmt.Fprintf(w, "%s", err)
                }
        }
}

func dockerHandler(containerManager manager.Manager) auth.AuthenticatedHandlerFunc {
        return func(w http.ResponseWriter, r *auth.AuthenticatedRequest) {
                err := serveDockerPage(containerManager, w, r.URL)
                if err != nil {
                        fmt.Fprintf(w, "%s", err)
                }
	}
}

// Register http handlers
func RegisterHandlersDigest(containerManager manager.Manager,authenticator *auth.DigestAuth) error {
	// Register the handler for the containers page.
	if authenticator!=nil {
		http.HandleFunc(ContainersPage, authenticator.Wrap(containerHandler(containerManager)))
                http.HandleFunc(DockerPage, authenticator.Wrap(dockerHandler(containerManager)))
	} else {
		http.HandleFunc(ContainersPage, containerHandlerNoAuth(containerManager))
                http.HandleFunc(DockerPage, dockerHandlerNoAuth(containerManager))
	}
	return nil
}

func RegisterHandlersBasic(containerManager manager.Manager,authenticator *auth.BasicAuth) error {
        // Register the handler for the containers and docker age.
        if authenticator!=nil {
                http.HandleFunc(ContainersPage, authenticator.Wrap(containerHandler(containerManager)))
                http.HandleFunc(DockerPage, authenticator.Wrap(dockerHandler(containerManager)))
        } else {
                http.HandleFunc(ContainersPage, containerHandlerNoAuth(containerManager))
                http.HandleFunc(DockerPage, dockerHandlerNoAuth(containerManager))
        }
        return nil
}


func getContainerDisplayName(cont info.ContainerReference) string {
	// Pick the shortest name of the container as the display name.
	displayName := cont.Name
	for _, alias := range cont.Aliases {
		if len(displayName) >= len(alias) {
			displayName = alias
		}
	}

	// Add the full container name to the display name.
	if displayName != cont.Name {
		displayName = fmt.Sprintf("%s (%s)", displayName, cont.Name)
	}

	return displayName
}
