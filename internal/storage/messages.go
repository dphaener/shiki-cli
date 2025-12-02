package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/dphaener/shiki-cli/pkg/types"
)

// MessageFrontmatter is the YAML header for message files
type MessageFrontmatter struct {
	From      string    `yaml:"from"`
	To        string    `yaml:"to"`
	Timestamp time.Time `yaml:"timestamp"`
	Turn      int       `yaml:"turn"`
}

// WriteMessage creates a message file
func WriteMessage(workspaceDir string, msg *types.Message) error {
	// Format: messages/00-agent_1-to-agent_2.md
	filename := fmt.Sprintf("%02d-%s-to-%s.md", msg.Turn, msg.From, msg.To)
	path := filepath.Join(workspaceDir, "messages", filename)

	// Build frontmatter
	frontmatter := MessageFrontmatter{
		From:      msg.From,
		To:        msg.To,
		Timestamp: msg.Timestamp,
		Turn:      msg.Turn,
	}

	yamlData, err := yaml.Marshal(frontmatter)
	if err != nil {
		return fmt.Errorf("marshal frontmatter: %w", err)
	}

	// Build full content
	content := fmt.Sprintf("---\n%s---\n\n%s", string(yamlData), msg.Content)

	if err := AtomicWriteString(path, content, 0o600); err != nil {
		return fmt.Errorf("write message file: %w", err)
	}

	msg.FilePath = path
	msg.ID = fmt.Sprintf("msg-%d-%s", msg.Turn, msg.From)

	return nil
}

// ReadMessages reads all messages from the other agent
func ReadMessages(workspaceDir, forAgent string, sinceTurn int) ([]*types.Message, error) {
	messagesDir := filepath.Join(workspaceDir, "messages")

	entries, err := os.ReadDir(messagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.Message{}, nil
		}
		return nil, fmt.Errorf("read messages dir: %w", err)
	}

	var messages []*types.Message
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(messagesDir, entry.Name())
		msg, err := parseMessageFile(path)
		if err != nil {
			continue // Skip malformed files
		}

		// Filter: only messages TO this agent, from sinceTurn onwards
		if msg.To == forAgent && msg.Turn >= sinceTurn {
			messages = append(messages, msg)
		}
	}

	// Sort by turn, then timestamp
	sort.Slice(messages, func(i, j int) bool {
		if messages[i].Turn != messages[j].Turn {
			return messages[i].Turn < messages[j].Turn
		}
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})

	return messages, nil
}

func parseMessageFile(path string) (*types.Message, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: Reading from validated path
	if err != nil {
		return nil, err
	}

	// Split frontmatter and content
	parts := strings.SplitN(string(data), "---\n", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid message format: missing frontmatter")
	}

	var fm MessageFrontmatter
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	content := strings.TrimSpace(parts[2])

	return &types.Message{
		ID:        fmt.Sprintf("msg-%d-%s", fm.Turn, fm.From),
		From:      fm.From,
		To:        fm.To,
		Timestamp: fm.Timestamp,
		Turn:      fm.Turn,
		Content:   content,
		FilePath:  path,
	}, nil
}
