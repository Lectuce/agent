package session

import (
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

type Session struct {
	ID              string
	Messages        []anthropic.MessageParam
	RoundsSinceTodo int
}

type SessionManager struct {
	Sessions map[string]*Session
	Current  string
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		Sessions: map[string]*Session{
			"default": {
				ID:       "default",
				Messages: []anthropic.MessageParam{},
			},
		},
		Current: "default",
	}
}

func (sm *SessionManager) NewSession(name string) bool {
	_, exists := sm.Sessions[name]
	if exists {
		return false
	}

	sm.Sessions[name] = &Session{
		ID:       name,
		Messages: []anthropic.MessageParam{},
	}
	sm.Current = name
	return true
}

func (sm *SessionManager) CurrentSession() *Session {
	return sm.Sessions[sm.Current]
}

func (sm *SessionManager) HandleCommand(query string) bool {

	// 新session
	if strings.HasPrefix(query, "/session new") {
		name := strings.TrimSpace(strings.TrimPrefix(query, "/session new"))
		if name == "" {
			fmt.Println("session name is required")
			return true
		}

		if !sm.NewSession(name) {
			fmt.Printf("session already exists: %s\n", name)
			return true
		}

		sm.Current = name
		fmt.Printf("created and switch to session: %s\n", name)
		return true
	}

	// 切换 session
	if strings.HasPrefix(query, "/session switch") {
		name := strings.TrimSpace(
			strings.TrimPrefix(
				query,
				"/session switch ",
			),
		)
		_, ok := sm.Sessions[name]
		if !ok {
			fmt.Printf(
				"session not found: %s\n",
				name,
			)
			return true
		}
		sm.Current = name
		fmt.Printf("switched to session: %s\n", name)
		return true
	}

	// 查看 session
	if query == "/session list" {
		for name := range sm.Sessions {
			if name == sm.Current {
				fmt.Printf("* %s\n", name)
			} else {
				fmt.Printf("  %s\n", name)
			}
		}
		return true
	}

	return false
}
