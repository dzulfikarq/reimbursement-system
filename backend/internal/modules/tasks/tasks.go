// Package tasks defines asynq task type names shared by the API (producer)
// and the worker (consumer). Docs/06: email.send, report.generate.
package tasks

const (
	TypeEmailSend = "email:send"
	TypeReportGen = "report:generate"
)
