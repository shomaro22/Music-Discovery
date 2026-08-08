package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"music-discovery/backend/entity"
)

var ErrAlertNotifierNotConfigured = errors.New("alert notifier is not configured")

type AlertNotifier interface {
	Notify(context.Context, AlertState, string) error
}

type AlertStatePort interface {
	List(context.Context) ([]entity.AlertStateRecord, error)
	FindByCode(context.Context, string) (*entity.AlertStateRecord, error)
	UpsertState(context.Context, *entity.AlertStateRecord) error
	MarkNotified(context.Context, string, string, time.Time) error
}

type webhookAlertNotifier struct {
	url    string
	client *http.Client
}

func NewWebhookAlertNotifier(url string, timeout time.Duration) AlertNotifier {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &webhookAlertNotifier{url: strings.TrimSpace(url), client: &http.Client{Timeout: timeout}}
}
func (n *webhookAlertNotifier) Notify(ctx context.Context, alert AlertState, transition string) error {
	if n.url == "" {
		return ErrAlertNotifierNotConfigured
	}
	body, err := json.Marshal(map[string]interface{}{"transition": transition, "alert": alert})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("alert webhook status: %d", res.StatusCode)
	}
	return nil
}

type IAlertMonitor interface {
	EvaluateAndPublish(context.Context) error
	ListCurrent(context.Context) ([]AlertState, error)
	SendTest(context.Context) error
}

type alertMonitor struct {
	alerts   IAlertUsecase
	health   IHealthUsecase
	notifier AlertNotifier
	states   AlertStatePort
}

func NewAlertMonitor(alerts IAlertUsecase, health IHealthUsecase, notifier AlertNotifier, ports ...AlertStatePort) IAlertMonitor {
	var port AlertStatePort
	if len(ports) > 0 {
		port = ports[0]
	}
	return &alertMonitor{alerts: alerts, health: health, notifier: notifier, states: port}
}

func (m *alertMonitor) EvaluateAndPublish(ctx context.Context) error {
	states := m.alerts.Evaluate(ctx, m.health.Readiness(ctx))
	now := time.Now().UTC()
	var resultErr error

	for _, state := range states {
		var previous *entity.AlertStateRecord
		if m.states != nil {
			stored, err := m.states.FindByCode(ctx, state.Code)
			if err != nil {
				resultErr = errors.Join(resultErr, err)
				continue
			}
			previous = stored
			if err := m.states.UpsertState(ctx, &entity.AlertStateRecord{
				Code: state.Code, Severity: state.Severity, Status: state.Status,
				Message: state.Message, Value: state.Value, Threshold: state.Threshold,
				UpdatedAt: now,
			}); err != nil {
				resultErr = errors.Join(resultErr, err)
				continue
			}
		}

		lastNotifiedStatus := ""
		if previous != nil {
			lastNotifiedStatus = previous.LastNotifiedStatus
		}
		if lastNotifiedStatus == state.Status {
			continue
		}
		if previous == nil && state.Status == "resolved" {
			continue
		}

		transition := "firing"
		if state.Status == "resolved" {
			transition = "recovered"
		}
		if err := m.notifier.Notify(ctx, state, transition); err != nil {
			// 通知先未設定でもAlert状態の評価・保存は継続する。
			// Test AlertはSendTestから未設定Errorをそのまま返す。
			if !errors.Is(err, ErrAlertNotifierNotConfigured) {
				resultErr = errors.Join(resultErr, err)
			}
			continue
		}
		if m.states != nil {
			if err := m.states.MarkNotified(ctx, state.Code, state.Status, now); err != nil {
				resultErr = errors.Join(resultErr, err)
			}
		}
	}
	return resultErr
}

func (m *alertMonitor) ListCurrent(ctx context.Context) ([]AlertState, error) {
	if m.states == nil {
		return []AlertState{}, nil
	}
	records, err := m.states.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]AlertState, 0, len(records))
	for _, r := range records {
		result = append(result, AlertState{Code: r.Code, Severity: r.Severity, Status: r.Status, Message: r.Message, Value: r.Value, Threshold: r.Threshold, CheckedAt: r.UpdatedAt})
	}
	return result, nil
}
func (m *alertMonitor) SendTest(ctx context.Context) error {
	return m.notifier.Notify(ctx, AlertState{Code: "TEST_ALERT", Severity: "info", Status: "firing", Message: "Monitoring notification test", CheckedAt: time.Now().UTC()}, "test")
}
