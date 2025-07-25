package status

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ProjectStatus struct {
	Status string `json:"status"` // "running" ou "stop"
}

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

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

var (
	currentStatus   = &ProjectStatus{Status: "stop"}
	projectSubs     = make(map[*websocket.Conn]chan struct{})
	projectSubsMu   sync.Mutex
	currentStatusMu sync.Mutex

	jobStatusMap  = make(map[string]*JobStatus)
	jobStatusMu   sync.Mutex
	subscribers   = make(map[*websocket.Conn]chan struct{})
	subscribersMu sync.Mutex

	logConns   = make(map[*websocket.Conn]chan struct{})
	logConnsMu sync.Mutex

	jobLogs   = make(map[string][]LogEntry)
	jobLogsMu sync.Mutex
)

func ClearJobLogs() {
	jobLogsMu.Lock()
	defer jobLogsMu.Unlock()
	jobLogs = make(map[string][]LogEntry)
}

func UpdateProjectStatus(status string) {
	currentStatusMu.Lock()
	currentStatus.Status = status
	currentStatusMu.Unlock()
	notifyProjectSubscribers()
}

func notifyProjectSubscribers() {
	projectSubsMu.Lock()
	defer projectSubsMu.Unlock()

	for _, ch := range projectSubs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

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

func AppendLog(message string) {
	jobLogsMu.Lock()
	defer jobLogsMu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Message:   message,
	}
	jobLogs["default"] = append(jobLogs["default"], entry)

	notifyLogSubscribers() // <- versão sem jobID
}

func notifyLogSubscribers() {
	logConnsMu.Lock()
	defer logConnsMu.Unlock()

	for _, ch := range logConns {
		select {
		case ch <- struct{}{}:
		default:
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

func ProjectStatusWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := make(chan struct{}, 1)
	projectSubsMu.Lock()
	projectSubs[conn] = ch
	projectSubsMu.Unlock()

	defer func() {
		projectSubsMu.Lock()
		delete(projectSubs, conn)
		projectSubsMu.Unlock()
	}()

	for range ch {
		currentStatusMu.Lock()
		statusCopy := *currentStatus
		currentStatusMu.Unlock()

		if data, err := json.Marshal(statusCopy); err == nil {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				break
			}
		}
	}
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

func LogsWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := make(chan struct{}, 1)

	logConnsMu.Lock()
	logConns[conn] = ch
	logConnsMu.Unlock()

	defer func() {
		logConnsMu.Lock()
		delete(logConns, conn)
		logConnsMu.Unlock()
	}()

	for range ch {
		jobLogsMu.Lock()
		logCopy := append([]LogEntry(nil), jobLogs["default"]...)
		jobLogsMu.Unlock()

		if data, err := json.Marshal(logCopy); err == nil {
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
