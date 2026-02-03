package jobrunner

import "sync"

var (
	activeRunnerMu sync.Mutex
	activeRunner   *JobRunner
)

func SetActiveRunner(runner *JobRunner) {
	activeRunnerMu.Lock()
	activeRunner = runner
	activeRunnerMu.Unlock()
}

func StopActiveRunner(projectID string, reason string) bool {
	activeRunnerMu.Lock()
	runner := activeRunner
	activeRunnerMu.Unlock()

	if runner == nil {
		return false
	}
	if projectID != "" && runner.ProjectID != projectID {
		return false
	}
	runner.Stop(reason)
	return true
}

func clearActiveRunner(runner *JobRunner) {
	activeRunnerMu.Lock()
	if activeRunner == runner {
		activeRunner = nil
	}
	activeRunnerMu.Unlock()
}
