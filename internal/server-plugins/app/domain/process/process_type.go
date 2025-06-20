package process

import "fmt"

type ProcessType string

const (
	ProcessTypeWeb     ProcessType = "web"
	ProcessTypeWorker  ProcessType = "worker"
	ProcessTypeCron    ProcessType = "cron"
	ProcessTypeRelease ProcessType = "release"
	ProcessTypeUtil    ProcessType = "util"
)

func NewProcessType(processType string) (ProcessType, error) {
	pt := ProcessType(processType)
	if !pt.IsValid() {
		return "", fmt.Errorf("invalid process type: %s", processType)
	}
	return pt, nil
}

func (pt ProcessType) IsValid() bool {
	validTypes := []ProcessType{
		ProcessTypeWeb, ProcessTypeWorker, ProcessTypeCron,
		ProcessTypeRelease, ProcessTypeUtil,
	}

	for _, validType := range validTypes {
		if pt == validType {
			return true
		}
	}
	return false
}

func (pt ProcessType) IsWebProcess() bool {
	return pt == ProcessTypeWeb
}

func (pt ProcessType) RequiresHTTPAccess() bool {
	return pt == ProcessTypeWeb
}

func (pt ProcessType) String() string {
	return string(pt)
}
