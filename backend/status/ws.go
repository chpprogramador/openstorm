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

type CountStatus struct {
	Done  int `json:"done"`
	Total int `json:"total"`
}

type WorkerStatus struct {
	ReadActive  int `json:"readActive"`
	ReadTotal   int `json:"readTotal"`
	WriteActive int `json:"writeActive"`
	WriteTotal  int `json:"writeTotal"`
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

	countStatus   = &CountStatus{}
	countSubs     = make(map[*websocket.Conn]chan struct{})
	countSubsMu   sync.Mutex
	countStatusMu sync.Mutex

	workerStatus   = &WorkerStatus{}
	workerSubs     = make(map[*websocket.Conn]chan struct{})
	workerSubsMu   sync.Mutex
	workerStatusMu sync.Mutex
)

func ClearJobLogs() {
	jobLogsMu.Lock()
	defer jobLogsMu.Unlock()
	jobLogs = make(map[string][]LogEntry)
}

func ResetCountStatus() {
	countStatusMu.Lock()
	countStatus.Done = 0
	countStatus.Total = 0
	countStatusMu.Unlock()
	notifyCountSubscribers()
}

func SetCountTotal(total int) {
	countStatusMu.Lock()
	countStatus.Total = total
	countStatusMu.Unlock()
	notifyCountSubscribers()
}

func IncCountDone() {
	countStatusMu.Lock()
	countStatus.Done++
	countStatusMu.Unlock()
	notifyCountSubscribers()
}

func ResetWorkerStatus() {
	workerStatusMu.Lock()
	workerStatus.ReadActive = 0
	workerStatus.ReadTotal = 0
	workerStatus.WriteActive = 0
	workerStatus.WriteTotal = 0
	workerStatusMu.Unlock()
	notifyWorkerSubscribers()
}

func AddWorkerTotals(readDelta, writeDelta int) {
	workerStatusMu.Lock()
	workerStatus.ReadTotal += readDelta
	if workerStatus.ReadTotal < 0 {
		workerStatus.ReadTotal = 0
	}
	workerStatus.WriteTotal += writeDelta
	if workerStatus.WriteTotal < 0 {
		workerStatus.WriteTotal = 0
	}
	workerStatusMu.Unlock()
	notifyWorkerSubscribers()
}

func AddWorkerActive(readDelta, writeDelta int) {
	workerStatusMu.Lock()
	workerStatus.ReadActive += readDelta
	if workerStatus.ReadActive < 0 {
		workerStatus.ReadActive = 0
	}
	workerStatus.WriteActive += writeDelta
	if workerStatus.WriteActive < 0 {
		workerStatus.WriteActive = 0
	}
	workerStatusMu.Unlock()
	notifyWorkerSubscribers()
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

func notifyCountSubscribers() {
	countSubsMu.Lock()
	defer countSubsMu.Unlock()

	for _, ch := range countSubs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func notifyWorkerSubscribers() {
	workerSubsMu.Lock()
	defer workerSubsMu.Unlock()

	for _, ch := range workerSubs {
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

func CountStatusWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := make(chan struct{}, 1)
	countSubsMu.Lock()
	countSubs[conn] = ch
	countSubsMu.Unlock()

	defer func() {
		countSubsMu.Lock()
		delete(countSubs, conn)
		countSubsMu.Unlock()
	}()

	// Envia o status inicial
	ch <- struct{}{}

	for range ch {
		countStatusMu.Lock()
		statusCopy := *countStatus
		countStatusMu.Unlock()

		if data, err := json.Marshal(statusCopy); err == nil {
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				break
			}
		}
	}
}

func WorkerStatusWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ch := make(chan struct{}, 1)
	workerSubsMu.Lock()
	workerSubs[conn] = ch
	workerSubsMu.Unlock()

	defer func() {
		workerSubsMu.Lock()
		delete(workerSubs, conn)
		workerSubsMu.Unlock()
	}()

	// Envia o status inicial
	ch <- struct{}{}

	for range ch {
		workerStatusMu.Lock()
		statusCopy := *workerStatus
		workerStatusMu.Unlock()

		if data, err := json.Marshal(statusCopy); err == nil {
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
