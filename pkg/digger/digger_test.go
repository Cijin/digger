package digger

import (
	"digger/pkg/configuration"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type RunInfo struct {
	Command   string
	Params    string
	Timestamp time.Time
}

type MockCommandRunner struct {
	Commands []RunInfo
}

func (m *MockCommandRunner) Run(command string) (string, string, error) {
	m.Commands = append(m.Commands, RunInfo{"Run", command, time.Now()})
	return "", "", nil
}

type MockTerraformExecutor struct {
	Commands []RunInfo
}

func (m *MockTerraformExecutor) Init(params []string) (string, string, error) {
	m.Commands = append(m.Commands, RunInfo{"Init", strings.Join(params, " "), time.Now()})
	return "", "", nil
}

func (m *MockTerraformExecutor) Apply(params []string, plan string) (string, string, error) {
	m.Commands = append(m.Commands, RunInfo{"Apply", strings.Join(params, " ") + " " + plan, time.Now()})
	return "", "", nil
}

func (m *MockTerraformExecutor) Plan(params []string) (bool, string, string, error) {
	m.Commands = append(m.Commands, RunInfo{"Plan", strings.Join(params, " "), time.Now()})
	return true, "", "", nil
}

type MockPRManager struct {
	Commands []RunInfo
}

func (m *MockPRManager) GetChangedFiles(prNumber int) ([]string, error) {
	m.Commands = append(m.Commands, RunInfo{"GetChangedFiles", strconv.Itoa(prNumber), time.Now()})
	return []string{}, nil
}

func (m *MockPRManager) PublishComment(prNumber int, comment string) {
	m.Commands = append(m.Commands, RunInfo{"PublishComment", strconv.Itoa(prNumber) + " " + comment, time.Now()})
}

func (m *MockPRManager) SetStatus(prNumber int, status string, statusContext string) error {
	m.Commands = append(m.Commands, RunInfo{"SetStatus", strconv.Itoa(prNumber) + " " + status + " " + statusContext, time.Now()})
	return nil
}

func (m *MockPRManager) GetCombinedPullRequestStatus(prNumber int) (string, error) {
	m.Commands = append(m.Commands, RunInfo{"GetCombinedPullRequestStatus", strconv.Itoa(prNumber), time.Now()})
	return "", nil
}

func (m *MockPRManager) MergePullRequest(prNumber int) error {
	m.Commands = append(m.Commands, RunInfo{"MergePullRequest", strconv.Itoa(prNumber), time.Now()})
	return nil
}

func (m *MockPRManager) IsMergeable(prNumber int) (bool, string, error) {
	m.Commands = append(m.Commands, RunInfo{"IsMergeable", strconv.Itoa(prNumber), time.Now()})
	return true, "", nil
}

func (m *MockPRManager) DownloadLatestPlans(prNumber int) (string, error) {
	m.Commands = append(m.Commands, RunInfo{"DownloadLatestPlans", strconv.Itoa(prNumber), time.Now()})
	return "plan", nil
}

type MockProjectLock struct {
	Commands []RunInfo
}

func (m *MockProjectLock) Lock(prNumber int) (bool, error) {
	m.Commands = append(m.Commands, RunInfo{"Lock", strconv.Itoa(prNumber), time.Now()})
	return true, nil
}

func (m *MockProjectLock) Unlock(prNumber int) (bool, error) {
	m.Commands = append(m.Commands, RunInfo{"Unlock", strconv.Itoa(prNumber), time.Now()})
	return true, nil
}

func (m *MockProjectLock) ForceUnlock(prNumber int) error {
	m.Commands = append(m.Commands, RunInfo{"ForceUnlock", strconv.Itoa(prNumber), time.Now()})
	return nil
}

func (m *MockProjectLock) LockId() string {
	m.Commands = append(m.Commands, RunInfo{"LockId", "", time.Now()})
	return ""
}

type MockZipper struct {
	Commands []RunInfo
}

func (m *MockZipper) GetFileFromZip(zipFile string, filename string) (string, error) {
	m.Commands = append(m.Commands, RunInfo{"GetFileFromZip", zipFile + " " + filename, time.Now()})
	return "plan", nil
}

func TestCorrectCommandExecutionWhenApplying(t *testing.T) {

	commandRunner := &MockCommandRunner{}
	terraformExecutor := &MockTerraformExecutor{}
	prManager := &MockPRManager{}
	lock := &MockProjectLock{}
	zipper := &MockZipper{}

	executor := DiggerExecutor{
		applyStage: &configuration.Stage{
			Steps: []configuration.Step{
				{
					Action:    "init",
					ExtraArgs: nil,
					Value:     "",
				},
				{
					Action:    "apply",
					ExtraArgs: nil,
					Value:     "",
				},
				{
					Action:    "run",
					ExtraArgs: nil,
					Value:     "echo",
				},
			},
		},
		planStage:         &configuration.Stage{},
		zipManager:        zipper,
		commandRunner:     commandRunner,
		terraformExecutor: terraformExecutor,
		prManager:         prManager,
		lock:              lock,
	}

	executor.Apply(1)

	commandStrings := allCommandsInOrderWithParams(terraformExecutor, commandRunner, prManager, lock)

	assert.Equal(t, []string{"DownloadLatestPlans 1", "IsMergeable 1", "Lock 1", "Init ", "Apply  plan", "LockId ", "PublishComment 1 <details>\n  <summary>Apply for ****</summary>\n\n  ```terraform\n\n  ```\n</details>", "Unlock 1", "Run echo", "LockId "}, commandStrings)
}

func TestCorrectCommandExecutionWhenPlanning(t *testing.T) {

	commandRunner := &MockCommandRunner{}
	terraformExecutor := &MockTerraformExecutor{}
	prManager := &MockPRManager{}
	lock := &MockProjectLock{}

	executor := DiggerExecutor{
		applyStage: &configuration.Stage{},
		planStage: &configuration.Stage{
			Steps: []configuration.Step{
				{
					Action:    "init",
					ExtraArgs: nil,
					Value:     "",
				},
				{
					Action:    "plan",
					ExtraArgs: nil,
					Value:     "",
				},
				{
					Action:    "run",
					ExtraArgs: nil,
					Value:     "echo",
				},
			},
		},
		commandRunner:     commandRunner,
		terraformExecutor: terraformExecutor,
		prManager:         prManager,
		lock:              lock,
	}

	executor.Plan(1)

	commandStrings := allCommandsInOrderWithParams(terraformExecutor, commandRunner, prManager, lock)

	assert.Equal(t, []string{"Lock 1", "Init ", "Plan -out .tfplan", "LockId ", "PublishComment 1 <details>\n  <summary>Plan for ****</summary>\n\n  ```terraform\n\n  ```\n</details>", "Run echo", "LockId "}, commandStrings)
}

func allCommandsInOrderWithParams(terraformExecutor *MockTerraformExecutor, commandRunner *MockCommandRunner, prManager *MockPRManager, lock *MockProjectLock) []string {
	var commands []RunInfo
	for _, command := range terraformExecutor.Commands {
		commands = append(commands, command)
	}
	for _, command := range commandRunner.Commands {
		commands = append(commands, command)
	}
	for _, command := range prManager.Commands {
		commands = append(commands, command)
	}
	for _, command := range lock.Commands {
		commands = append(commands, command)
	}

	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Timestamp.Before(commands[j].Timestamp)
	})

	// turn commands into string slice join command and it's arguments into a string
	var commandStrings []string
	for _, command := range commands {
		commandStrings = append(commandStrings, command.Command+" "+command.Params)
	}
	return commandStrings
}

func TestParseWorkspace(t *testing.T) {
	var commentTests = []struct {
		in  string
		out string
		err bool
	}{
		{"test", "", false},
		{"test -w workspace", "workspace", false},
		{"test -w workspace -w workspace2", "", true},
		{"test -w", "", true},
	}

	for _, tt := range commentTests {
		out, err := parseWorkspace(tt.in)
		if tt.err {
			if err == nil {
				t.Errorf("parseWorkspace(%q) = %q, want error", tt.in, out)
			}
		} else {
			if err != nil {
				t.Errorf("parseWorkspace(%q) = %q, want %q", tt.in, err, tt.out)
			}
			if out != tt.out {
				t.Errorf("parseWorkspace(%q) = %q, want %q", tt.in, out, tt.out)
			}
		}
	}

}