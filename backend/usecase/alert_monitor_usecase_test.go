package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"music-discovery/backend/entity"
)

type alertNotifierStub struct {
	mu    sync.Mutex
	err   error
	calls []string
}

func (n *alertNotifierStub) Notify(_ context.Context, _ AlertState, transition string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.calls = append(n.calls, transition)
	return n.err
}

type alertStatePortStub struct {
	values            map[string]*entity.AlertStateRecord
	upserts, notified int
}

func (p *alertStatePortStub) List(context.Context) ([]entity.AlertStateRecord, error) {
	result := []entity.AlertStateRecord{}
	for _, v := range p.values {
		result = append(result, *v)
	}
	return result, nil
}
func (p *alertStatePortStub) FindByCode(_ context.Context, code string) (*entity.AlertStateRecord, error) {
	return p.values[code], nil
}
func (p *alertStatePortStub) UpsertState(_ context.Context, value *entity.AlertStateRecord) error {
	p.upserts++
	current := p.values[value.Code]
	if current == nil {
		current = &entity.AlertStateRecord{Code: value.Code}
	}
	current.Status = value.Status
	current.Severity = value.Severity
	current.Message = value.Message
	current.Value = value.Value
	current.Threshold = value.Threshold
	current.UpdatedAt = value.UpdatedAt
	p.values[value.Code] = current
	return nil
}
func (p *alertStatePortStub) MarkNotified(_ context.Context, code, status string, at time.Time) error {
	p.notified++
	p.values[code].LastNotifiedStatus = status
	p.values[code].LastNotifiedAt = &at
	return nil
}

type alertUsecaseStub struct{ states []AlertState }

func (s alertUsecaseStub) Evaluate(context.Context, ReadinessResult) []AlertState { return s.states }

type healthUsecaseStub struct{}

func (healthUsecaseStub) Readiness(context.Context) ReadinessResult {
	return ReadinessResult{Status: "ready"}
}

func TestAlertMonitorRetriesAfterNotificationFailure(t *testing.T) {
	port := &alertStatePortStub{values: map[string]*entity.AlertStateRecord{"A": {Code: "A", Status: "resolved", LastNotifiedStatus: "resolved"}}}
	notifier := &alertNotifierStub{err: errors.New("failed")}
	monitor := NewAlertMonitor(alertUsecaseStub{[]AlertState{{Code: "A", Status: "firing"}}}, healthUsecaseStub{}, notifier, port)
	if err := monitor.EvaluateAndPublish(context.Background()); err == nil {
		t.Fatal("expected notification error")
	}
	if port.values["A"].LastNotifiedStatus != "resolved" {
		t.Fatal("notified status must not advance on failure")
	}
	notifier.err = nil
	if err := monitor.EvaluateAndPublish(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(notifier.calls) != 2 {
		t.Fatalf("calls=%d", len(notifier.calls))
	}
	if port.values["A"].LastNotifiedStatus != "firing" {
		t.Fatal("notification success was not recorded")
	}
}

func TestAlertMonitorSuppressesDuplicateAndSendsRecovery(t *testing.T) {
	port := &alertStatePortStub{values: map[string]*entity.AlertStateRecord{"A": {Code: "A", Status: "firing", LastNotifiedStatus: "firing"}}}
	notifier := &alertNotifierStub{}
	monitor := NewAlertMonitor(alertUsecaseStub{[]AlertState{{Code: "A", Status: "firing"}}}, healthUsecaseStub{}, notifier, port)
	if err := monitor.EvaluateAndPublish(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(notifier.calls) != 0 {
		t.Fatal("duplicate firing notification")
	}
	monitor = NewAlertMonitor(alertUsecaseStub{[]AlertState{{Code: "A", Status: "resolved"}}}, healthUsecaseStub{}, notifier, port)
	if err := monitor.EvaluateAndPublish(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(notifier.calls) != 1 || notifier.calls[0] != "recovered" {
		t.Fatalf("calls=%v", notifier.calls)
	}
}

func TestWebhookNotifierRequiresConfiguration(t *testing.T) {
	notifier := NewWebhookAlertNotifier("", time.Second)
	err := notifier.Notify(context.Background(), AlertState{}, "test")
	if !errors.Is(err, ErrAlertNotifierNotConfigured) {
		t.Fatalf("err=%v", err)
	}
}

func TestListCurrentHasNoPublishSideEffects(t *testing.T) {
	port := &alertStatePortStub{values: map[string]*entity.AlertStateRecord{"A": {Code: "A", Status: "firing", UpdatedAt: time.Now()}}}
	notifier := &alertNotifierStub{}
	monitor := NewAlertMonitor(alertUsecaseStub{}, healthUsecaseStub{}, notifier, port)
	states, err := monitor.ListCurrent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 1 {
		t.Fatalf("states=%d", len(states))
	}
	if len(notifier.calls) != 0 || port.upserts != 0 || port.notified != 0 {
		t.Fatal("ListCurrent caused side effects")
	}
}

func TestAlertMonitorContinuesAfterNotificationFailure(t *testing.T) {
	port := &alertStatePortStub{values: map[string]*entity.AlertStateRecord{
		"A": {Code: "A", Status: "resolved", LastNotifiedStatus: "resolved"},
		"B": {Code: "B", Status: "resolved", LastNotifiedStatus: "resolved"},
	}}
	notifier := &sequenceAlertNotifierStub{errors: []error{errors.New("first notification failed"), nil}}
	monitor := NewAlertMonitor(alertUsecaseStub{[]AlertState{
		{Code: "A", Status: "firing"},
		{Code: "B", Status: "firing"},
	}}, healthUsecaseStub{}, notifier, port)

	if err := monitor.EvaluateAndPublish(context.Background()); err == nil {
		t.Fatal("expected aggregated notification error")
	}
	if port.values["A"].Status != "firing" || port.values["B"].Status != "firing" {
		t.Fatalf("all alert states must be saved: A=%s B=%s", port.values["A"].Status, port.values["B"].Status)
	}
	if port.values["A"].LastNotifiedStatus != "resolved" {
		t.Fatal("failed notification must not advance notified status")
	}
	if port.values["B"].LastNotifiedStatus != "firing" {
		t.Fatal("later alert must still be notified")
	}
	if len(notifier.calls) != 2 {
		t.Fatalf("notification calls=%d", len(notifier.calls))
	}
}

func TestAlertMonitorContinuesWhenNotifierIsNotConfigured(t *testing.T) {
	port := &alertStatePortStub{values: map[string]*entity.AlertStateRecord{}}
	notifier := &alertNotifierStub{err: ErrAlertNotifierNotConfigured}
	monitor := NewAlertMonitor(alertUsecaseStub{[]AlertState{
		{Code: "A", Status: "firing"},
		{Code: "B", Status: "firing"},
	}}, healthUsecaseStub{}, notifier, port)

	if err := monitor.EvaluateAndPublish(context.Background()); err != nil {
		t.Fatalf("not configured must not fail periodic evaluation: %v", err)
	}
	if port.upserts != 2 {
		t.Fatalf("upserts=%d", port.upserts)
	}
	if port.notified != 0 {
		t.Fatalf("notified=%d", port.notified)
	}
}

type sequenceAlertNotifierStub struct {
	mu     sync.Mutex
	errors []error
	calls  []string
}

func (n *sequenceAlertNotifierStub) Notify(_ context.Context, _ AlertState, transition string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.calls = append(n.calls, transition)
	index := len(n.calls) - 1
	if index < len(n.errors) {
		return n.errors[index]
	}
	return nil
}
