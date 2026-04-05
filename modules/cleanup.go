package modules

import (
	"sync"
	"time"

	"github.com/amarnathcjd/gogram/telegram"
)

type ScheduledDeletion struct {
	Client    *telegram.Client
	ChatID    int64
	MessageID int32
	DeleteAt  time.Time
}

type deletionKey struct {
	chatID    int64
	messageID int32
}

type CleanupScheduler struct {
	mu        sync.RWMutex
	deletions map[deletionKey]*ScheduledDeletion
	stopChan  chan struct{}
}

var globalScheduler *CleanupScheduler

func init() {
	globalScheduler = &CleanupScheduler{
		deletions: make(map[deletionKey]*ScheduledDeletion),
		stopChan:  make(chan struct{}),
	}
	go globalScheduler.Run()
}

func (cs *CleanupScheduler) Schedule(
	client *telegram.Client,
	chatID int64,
	messageID int32,
	delay time.Duration,
) {
	key := deletionKey{chatID, messageID}

	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.deletions[key] = &ScheduledDeletion{
		Client:    client,
		ChatID:    chatID,
		MessageID: messageID,
		DeleteAt:  time.Now().Add(delay),
	}
}

func (cs *CleanupScheduler) Cancel(chatID int64, messageID int32) {
	key := deletionKey{chatID, messageID}

	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.deletions, key)
}

func (cs *CleanupScheduler) Run() {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cs.processScheduled()
		case <-cs.stopChan:
			return
		}
	}
}

func (cs *CleanupScheduler) processScheduled() {
	now := time.Now()
	var toDelete []deletionKey
	var deletions []*ScheduledDeletion

	// Collect deletions that are ready
	cs.mu.RLock()
	for key, deletion := range cs.deletions {
		if now.After(deletion.DeleteAt) || now.Equal(deletion.DeleteAt) {
			toDelete = append(toDelete, key)
			deletions = append(deletions, deletion)
		}
	}
	cs.mu.RUnlock()

	// Remove from map
	if len(toDelete) > 0 {
		cs.mu.Lock()
		for _, key := range toDelete {
			delete(cs.deletions, key)
		}
		cs.mu.Unlock()
	}

	// Group deletions by chat for batch processing
	chatMessages := make(map[int64][]int32)
	for _, deletion := range deletions {
		chatMessages[deletion.ChatID] = append(
			chatMessages[deletion.ChatID],
			deletion.MessageID,
		)
	}

	// Delete messages in batches
	for chatID, msgIDs := range chatMessages {
		if len(msgIDs) == 0 {
			continue
		}

		// Get client from first deletion
		var client *telegram.Client
		for _, deletion := range deletions {
			if deletion.ChatID == chatID {
				client = deletion.Client
				break
			}
		}

		if client == nil {
			continue
		}

		// Delete in batches of 100 (Telegram limit)
		batchSize := 50
		for i := 0; i < len(msgIDs); i += batchSize {
			end := i + batchSize
			if end > len(msgIDs) {
				end = len(msgIDs)
			}

			batch := msgIDs[i:end]
			go func(c *telegram.Client, cid int64, msgs []int32) {
				_, _ = c.DeleteMessages(cid, msgs)
			}(client, chatID, batch)
		}
	}
}

func (cs *CleanupScheduler) Stop() {
	close(cs.stopChan)
}

func (cs *CleanupScheduler) Count() int {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return len(cs.deletions)
}

// Helper function to schedule a message for deletion
func ScheduleMessageDeletion(
	client *telegram.Client,
	chatID int64,
	messageID int32,
	delay time.Duration,
) {
	globalScheduler.Schedule(client, chatID, messageID, delay)
}

// Helper function to cancel a scheduled deletion
func CancelMessageDeletion(chatID int64, messageID int32) {
	globalScheduler.Cancel(chatID, messageID)
}

// ScheduleDeletion schedules a message for deletion
func ScheduleDeletion(m *telegram.NewMessage, delay time.Duration) {
	if m == nil {
		return
	}
	ScheduleMessageDeletion(m.Client, m.ChatID(), int32(m.ID), delay)
}
