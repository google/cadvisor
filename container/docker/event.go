package docker

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	dclient "github.com/docker/docker/client"
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"k8s.io/klog"
)

const eventRestart = "restart"

type eventHandler struct {
	client           *dclient.Client
	containerHandler *containerlibcontainer.Handler
	eventMutex       sync.Mutex
	filter           filters.Args
	id               string
	// Tells the eventHandler to stop.
	stopChan chan struct{}
}

type worker struct {
	message events.Message
}

func newDockerEventHandler(dockerClient *dclient.Client, id string, containerHandler *containerlibcontainer.Handler) (*eventHandler, error) {
	filter := filters.NewArgs()
	filter.Add("type", events.ContainerEventType)
	filter.Add("event", eventRestart)
	filter.Add("container", id)

	return &eventHandler{
		client:           dockerClient,
		containerHandler: containerHandler,
		filter:           filter,
		id:               id,
		stopChan:         make(chan struct{}, 1),
	}, nil
}

func (eh *eventHandler) routeEvents(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	messages, errs := dockerClient.Events(ctx, types.EventsOptions{
		Filters: eh.filter,
	})
	for {
		select {
		case event := <-messages:
			message := worker{
				message: event,
			}
			message.processEvent(event, eh)
		case err := <-errs:
			return true, err
		case <-eh.stopChan:
			return false, nil
		}

	}
}

func (w worker) processEvent(event events.Message, eh *eventHandler) {
	if event.Action == eventRestart {
		eh.eventMutex.Lock()
		defer eh.eventMutex.Unlock()

		klog.V(6).Infof("event received for %s: %s", eh.id, event.Action)
		ctnr, err := eh.client.ContainerInspect(context.Background(), eh.id)
		if err != nil {
			klog.V(6).Infof("error while retrieving container info %s", err)
			return
		}

		eh.containerHandler.UpdatePid(ctnr.State.Pid)
	}
}

func (eh *eventHandler) EventMutex() *sync.Mutex {
	return &eh.eventMutex
}

func (eh *eventHandler) Start() {
	klog.V(6).Infof("started event monitoring on container %s", eh.id)

	go func() {
		for {
			restart, err := eh.routeEvents(context.Background())
			if err != nil {
				klog.V(6).Infof("error in event channel %v", err)
			}

			if !restart {
				break
			}
		}
	}()
}

func (eh *eventHandler) Stop() {
	klog.V(6).Infof("stopped event monitoring on container %s", eh.id)
	close(eh.stopChan)
}