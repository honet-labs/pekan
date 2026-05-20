package worker

import (
	"context"
	"fmt"
	"time"

	reminderdomain "pekan/backend/internal/modules/finance/reminders/domain"
	notificationdomain "pekan/backend/internal/modules/finance/notifications/domain"
)

type ReminderRepository interface {
	ListDueForProcessing(ctx context.Context, limit int) ([]reminderdomain.Reminder, error)
	MarkTriggered(ctx context.Context, tenantID, reminderID string, nextDueDate *time.Time, actorUserID string) error
}

type NotificationRepository interface {
	Create(ctx context.Context, in notificationdomain.Notification) (notificationdomain.Notification, error)
}

type Service struct {
	reminders    ReminderRepository
	notifications NotificationRepository
	pollInterval time.Duration
}

func NewService(reminderRepo ReminderRepository, notificationRepo NotificationRepository, pollInterval time.Duration) *Service {
	return &Service{
		reminders:    reminderRepo,
		notifications: notificationRepo,
		pollInterval: pollInterval,
	}
}

func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := s.Process(ctx); err != nil {
				fmt.Printf("reminder worker error: %v\n", err)
			}
		}
	}
}

func (s *Service) Process(ctx context.Context) error {
	items, err := s.reminders.ListDueForProcessing(ctx, 50)
	if err != nil {
		return err
	}
	for _, item := range items {
		title := fmt.Sprintf("Reminder due: %s", item.Title)
		message := fmt.Sprintf("Reminder %q is due on %s", item.Title, item.DueDate.Format("2006-01-02"))
		_, _ = s.notifications.Create(ctx, notificationdomain.Notification{
			TenantID:         item.TenantID,
			NotificationType: "reminder",
			Title:            title,
			Message:          message,
			Status:           "unread",
			CreatedBy:        item.CreatedBy,
			CreatedAt:        time.Now().UTC(),
		})

		nextDue := computeNextDueDate(item)
		_ = s.reminders.MarkTriggered(ctx, item.TenantID, item.ID, nextDue, item.UpdatedBy)
	}
	return nil
}

func computeNextDueDate(reminder reminderdomain.Reminder) *time.Time {
	switch reminder.RepeatInterval {
	case "daily":
		next := reminder.DueDate.AddDate(0, 0, 1)
		return &next
	case "weekly":
		next := reminder.DueDate.AddDate(0, 0, 7)
		return &next
	case "monthly":
		next := reminder.DueDate.AddDate(0, 1, 0)
		return &next
	default:
		return nil
	}
}

