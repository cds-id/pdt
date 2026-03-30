package telegram

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// streamWriter implements agent.StreamWriter for Telegram.
type streamWriter struct {
	bot    *tgbotapi.BotAPI
	chatID int64

	mu             sync.Mutex
	thinkingMsgID  int
	toolStatuses   []toolStatus
	contentBuf     strings.Builder
	lastEditTime   time.Time
	pendingEdit    bool
}

type toolStatus struct {
	Name   string
	Status string // "executing" or "completed"
}

func newStreamWriter(bot *tgbotapi.BotAPI, chatID int64) *streamWriter {
	return &streamWriter{
		bot:    bot,
		chatID: chatID,
	}
}

func (w *streamWriter) WriteThinking(message string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.thinkingMsgID == 0 {
		msg := tgbotapi.NewMessage(w.chatID, "⏳ "+message)
		sent, err := w.bot.Send(msg)
		if err != nil {
			return fmt.Errorf("send thinking: %w", err)
		}
		w.thinkingMsgID = sent.MessageID
		w.lastEditTime = time.Now()
		return nil
	}

	return w.editThinkingLocked()
}

func (w *streamWriter) WriteToolStatus(toolName, status string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	found := false
	for i, ts := range w.toolStatuses {
		if ts.Name == toolName {
			w.toolStatuses[i].Status = status
			found = true
			break
		}
	}
	if !found {
		w.toolStatuses = append(w.toolStatuses, toolStatus{Name: toolName, Status: status})
	}

	if time.Since(w.lastEditTime) < time.Second {
		w.pendingEdit = true
		return nil
	}

	return w.editThinkingLocked()
}

func (w *streamWriter) WriteContent(content string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.contentBuf.WriteString(content)
	return nil
}

func (w *streamWriter) WriteDone() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Flush any throttled pending edit
	if w.pendingEdit && w.thinkingMsgID != 0 {
		w.editThinkingLocked()
	}

	// Final edit on thinking message
	if w.thinkingMsgID != 0 {
		names := make([]string, 0, len(w.toolStatuses))
		for _, ts := range w.toolStatuses {
			names = append(names, ts.Name)
		}
		var text string
		if len(names) > 0 {
			text = "✅ Done (used: " + strings.Join(names, ", ") + ")"
		} else {
			text = "✅ Done"
		}
		edit := tgbotapi.NewEditMessageText(w.chatID, w.thinkingMsgID, text)
		w.bot.Send(edit)
	}

	// Send final content as new message(s)
	content := strings.TrimSpace(w.contentBuf.String())
	if content == "" {
		return nil
	}

	chunks := splitMessage(content, 4096)
	for i, chunk := range chunks {
		msg := tgbotapi.NewMessage(w.chatID, escapeMarkdownV2(chunk))
		msg.ParseMode = "MarkdownV2"
		if _, err := w.bot.Send(msg); err != nil {
			// Retry without markdown on parse failure
			msg.ParseMode = ""
			msg.Text = chunk
			if _, err := w.bot.Send(msg); err != nil {
				return fmt.Errorf("send chunk %d: %w", i, err)
			}
		}
		if i < len(chunks)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

func (w *streamWriter) WriteError(msg string) error {
	w.mu.Lock()
	// Flush any pending thinking edits
	if w.pendingEdit && w.thinkingMsgID != 0 {
		w.editThinkingLocked()
	}
	w.mu.Unlock()

	errMsg := tgbotapi.NewMessage(w.chatID, "❌ Error: "+msg)
	_, err := w.bot.Send(errMsg)
	return err
}

func (w *streamWriter) editThinkingLocked() error {
	if w.thinkingMsgID == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("⏳ Processing...\n")
	for _, ts := range w.toolStatuses {
		if ts.Status == "completed" {
			sb.WriteString(fmt.Sprintf("\n✅ %s — done", ts.Name))
		} else {
			sb.WriteString(fmt.Sprintf("\n🔧 %s... %s", ts.Name, ts.Status))
		}
	}

	edit := tgbotapi.NewEditMessageText(w.chatID, w.thinkingMsgID, sb.String())
	_, err := w.bot.Send(edit)
	w.lastEditTime = time.Now()
	w.pendingEdit = false
	return err
}

// escapeMarkdownV2 escapes special characters for Telegram MarkdownV2 parse mode,
// preserving code blocks and bold formatting.
func escapeMarkdownV2(text string) string {
	parts := strings.Split(text, "```")

	var result strings.Builder
	for i, part := range parts {
		if i%2 == 1 {
			result.WriteString("```")
			result.WriteString(part)
			result.WriteString("```")
			continue
		}

		// Escape special MarkdownV2 chars, preserving * (bold) and _ (italic)
		// Backslash must be escaped first to avoid double-escaping
		escaped := strings.ReplaceAll(part, "\\", "\\\\")
		specialChars := []string{"[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
		for _, ch := range specialChars {
			escaped = strings.ReplaceAll(escaped, ch, "\\"+ch)
		}
		result.WriteString(escaped)
	}

	return result.String()
}

// splitMessage splits text into chunks of at most maxLen characters,
// preferring paragraph then line boundaries.
func splitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= maxLen {
			chunks = append(chunks, remaining)
			break
		}

		chunk := remaining[:maxLen]
		if idx := strings.LastIndex(chunk, "\n\n"); idx > 0 {
			chunks = append(chunks, remaining[:idx])
			remaining = remaining[idx+2:]
			continue
		}

		if idx := strings.LastIndex(chunk, "\n"); idx > 0 {
			chunks = append(chunks, remaining[:idx])
			remaining = remaining[idx+1:]
			continue
		}

		chunks = append(chunks, chunk)
		remaining = remaining[maxLen:]
	}

	return chunks
}
