package task

import (
	"github.com/sllt/pi-layout/internal/repository"
	"github.com/sllt/pi-layout/pkg/log"
	"github.com/sllt/pi-layout/pkg/sid"
)

type Task struct {
	logger *log.Logger
	sid    *sid.Sid
	tm     repository.Transaction
}

func NewTask(
	tm repository.Transaction,
	logger *log.Logger,
	sid *sid.Sid,
) *Task {
	return &Task{
		logger: logger,
		sid:    sid,
		tm:     tm,
	}
}
