package status

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type JobStatus struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Total     int        `json:"total"`
	Processed int        `json:"processed"`
	Progress  float64    `json:"progress"`
	Status    string     `json:"status"` // pending, running, done, error
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
	Error     string     `json:"error,omitempty"`
}

var (
	jobStatusMap  = make(map[string]*JobStatus)
	jobStatusMu   sync.Mutex
	subscribers   = make(map[*websocket.Conn]chan struct{})
	subscribersMu sync.Mutex
)

func NotifySubscribers() {
	subscribersMu.Lock()
	defer subscribersMu.Unlock()
	for _, ch := range subscribers {
		select {
		case ch <- struct{}{}:
		default: // Não bloquear se o canal já estiver cheio
		}
	}
}

func UpdateJobStatus(id string, update func(*JobStatus)) {
	jobStatusMu.Lock()
	defer jobStatusMu.Unlock()
	if _, ok := jobStatusMap[id]; !ok {
		jobStatusMap[id] = &JobStatus{ID: id, Status: "pending"}
	}
	update(jobStatusMap[id])
}

func GetAllJobStatus() []*JobStatus {
	jobStatusMu.Lock()
	defer jobStatusMu.Unlock()
	all := []*JobStatus{}
	for _, js := range jobStatusMap {
		copy := *js
		all = append(all, &copy)
	}
	return all
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func JobStatusWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := make(chan struct{}, 1)
	subscribersMu.Lock()
	subscribers[conn] = ch
	subscribersMu.Unlock()

	defer func() {
		subscribersMu.Lock()
		delete(subscribers, conn)
		subscribersMu.Unlock()
	}()

	for range ch {
		statuses := GetAllJobStatus()
		if data, err := json.Marshal(statuses); err == nil {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				break
			}
		}
	}
}

func GetJobStatus(id string) *JobStatus {
	jobStatusMu.Lock()
	defer jobStatusMu.Unlock()
	return jobStatusMap[id]
}
