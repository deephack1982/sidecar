package conversations

import (
	"testing"
	"time"

	"github.com/marcus/sidecar/internal/adapter"
)

func TestGroupMessagesIntoTurns(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		messages []adapter.Message
		want     []Turn
	}{
		{
			name:     "Empty messages",
			messages: []adapter.Message{},
			want:     nil,
		},
		{
			name: "Single user message",
			messages: []adapter.Message{
				{Role: "user", Content: "Hello", Timestamp: now},
			},
			want: []Turn{
				{
					Role:       "user",
					Messages:   []adapter.Message{{Role: "user", Content: "Hello", Timestamp: now}},
					StartIndex: 0,
				},
			},
		},
		{
			name: "Alternating roles",
			messages: []adapter.Message{
				{Role: "user", Content: "Hi", Timestamp: now},
				{Role: "assistant", Content: "Hello", Timestamp: now},
			},
			want: []Turn{
				{
					Role:       "user",
					Messages:   []adapter.Message{{Role: "user", Content: "Hi", Timestamp: now}},
					StartIndex: 0,
				},
				{
					Role:       "assistant",
					Messages:   []adapter.Message{{Role: "assistant", Content: "Hello", Timestamp: now}},
					StartIndex: 1,
				},
			},
		},
		{
			name: "Consecutive same role messages",
			messages: []adapter.Message{
				{Role: "user", Content: "Command output 1", Timestamp: now},
				{Role: "user", Content: "Command output 2", Timestamp: now},
				{Role: "user", Content: "Actual question", Timestamp: now},
			},
			want: []Turn{
				{
					Role: "user",
					Messages: []adapter.Message{
						{Role: "user", Content: "Command output 1", Timestamp: now},
						{Role: "user", Content: "Command output 2", Timestamp: now},
						{Role: "user", Content: "Actual question", Timestamp: now},
					},
					StartIndex: 0,
				},
			},
		},
		{
			name: "Mixed complex case",
			messages: []adapter.Message{
				{Role: "user", Content: "Q1", Timestamp: now, TokenUsage: adapter.TokenUsage{InputTokens: 10}},
				{Role: "assistant", Content: "A1 part 1", Timestamp: now, TokenUsage: adapter.TokenUsage{OutputTokens: 20}},
				{Role: "assistant", Content: "A1 part 2", Timestamp: now, TokenUsage: adapter.TokenUsage{OutputTokens: 15}, ToolUses: []adapter.ToolUse{{}}},
				{Role: "user", Content: "Q2", Timestamp: now, TokenUsage: adapter.TokenUsage{InputTokens: 5}},
			},
			want: []Turn{
				{
					Role:           "user",
					Messages:       []adapter.Message{{Role: "user", Content: "Q1", Timestamp: now, TokenUsage: adapter.TokenUsage{InputTokens: 10}}},
					StartIndex:     0,
					TotalTokensIn:  10,
					TotalTokensOut: 0,
				},
				{
					Role: "assistant",
					Messages: []adapter.Message{
						{Role: "assistant", Content: "A1 part 1", Timestamp: now, TokenUsage: adapter.TokenUsage{OutputTokens: 20}},
						{Role: "assistant", Content: "A1 part 2", Timestamp: now, TokenUsage: adapter.TokenUsage{OutputTokens: 15}, ToolUses: []adapter.ToolUse{{}}},
					},
					StartIndex:     1,
					TotalTokensIn:  0,
					TotalTokensOut: 35,
					ToolCount:      1,
				},
				{
					Role:           "user",
					Messages:       []adapter.Message{{Role: "user", Content: "Q2", Timestamp: now, TokenUsage: adapter.TokenUsage{InputTokens: 5}}},
					StartIndex:     3,
					TotalTokensIn:  5,
					TotalTokensOut: 0,
				},
			},
		},
		{
			name: "Token counting",
			messages: []adapter.Message{
				{
					Role: "assistant",
					ThinkingBlocks: []adapter.ThinkingBlock{
						{TokenCount: 100},
						{TokenCount: 50},
					},
					TokenUsage: adapter.TokenUsage{
						InputTokens:  10,
						OutputTokens: 20,
					},
				},
			},
			want: []Turn{
				{
					Role: "assistant",
					Messages: []adapter.Message{{
						Role: "assistant",
						ThinkingBlocks: []adapter.ThinkingBlock{
							{TokenCount: 100},
							{TokenCount: 50},
						},
						TokenUsage: adapter.TokenUsage{
							InputTokens:  10,
							OutputTokens: 20,
						},
					}},
					StartIndex:     0,
					TotalTokensIn:  10,
					TotalTokensOut: 20,
					ThinkingTokens: 150,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupMessagesIntoTurns(tt.messages)

			if len(got) != len(tt.want) {
				t.Fatalf("got %d turns, want %d", len(got), len(tt.want))
			}

			for i := range got {
				g := got[i]
				w := tt.want[i]

				if g.Role != w.Role {
					t.Errorf("turn[%d].Role = %q, want %q", i, g.Role, w.Role)
				}
				if g.StartIndex != w.StartIndex {
					t.Errorf("turn[%d].StartIndex = %d, want %d", i, g.StartIndex, w.StartIndex)
				}
				if len(g.Messages) != len(w.Messages) {
					t.Errorf("turn[%d].Messages length = %d, want %d", i, len(g.Messages), len(w.Messages))
				}
				if g.TotalTokensIn != w.TotalTokensIn {
					t.Errorf("turn[%d].TotalTokensIn = %d, want %d", i, g.TotalTokensIn, w.TotalTokensIn)
				}
				if g.TotalTokensOut != w.TotalTokensOut {
					t.Errorf("turn[%d].TotalTokensOut = %d, want %d", i, g.TotalTokensOut, w.TotalTokensOut)
				}
				if g.ThinkingTokens != w.ThinkingTokens {
					t.Errorf("turn[%d].ThinkingTokens = %d, want %d", i, g.ThinkingTokens, w.ThinkingTokens)
				}
				if g.ToolCount != w.ToolCount {
					t.Errorf("turn[%d].ToolCount = %d, want %d", i, g.ToolCount, w.ToolCount)
				}
			}
		})
	}
}

func TestStripXMLTags(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "Hello world",
			want:  "Hello world",
		},
		{
			input: "<tag>content</tag>",
			want:  "content",
		},
		{
			input: "Multiple <t1>tags</t1> here <t2>and</t2> there",
			want:  "Multiple tags here and there",
		},
		{
			input: "<user_query>Specific query</user_query>",
			want:  "Specific query",
		},
		{
			input: "Prefix <user_query>Query</user_query> Suffix",
			want:  "Query",
		},
		{
			input: "Mixed <other>tags</other> with <user_query>Target</user_query>",
			want:  "Target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripXMLTags(tt.input)
			if got != tt.want {
				t.Errorf("stripXMLTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTurnPreview(t *testing.T) {
	tests := []struct {
		name     string
		messages []adapter.Message
		maxLen   int
		want     string
	}{
		{
			name:     "Empty turn",
			messages: nil,
			maxLen:   10,
			want:     "",
		},
		{
			name: "Simple content",
			messages: []adapter.Message{
				{Content: "Hello world"},
			},
			maxLen: 20,
			want:   "Hello world",
		},
		{
			name: "Truncated content",
			messages: []adapter.Message{
				{Content: "Hello world"},
			},
			maxLen: 8,
			want:   "Hello...",
		},
		{
			name: "Skip tool result marker",
			messages: []adapter.Message{
				{Content: "[1 tool result(s)]"},
				{Content: "Actual content"},
			},
			maxLen: 20,
			want:   "Actual content",
		},
		{
			name: "XML tag stripping",
			messages: []adapter.Message{
				{Content: "<ant_thinking>Thinking...</ant_thinking>Response"},
			},
			maxLen: 20,
			want:   "Thinking...Response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			turn := Turn{Messages: tt.messages}
			got := turn.Preview(tt.maxLen)
			if got != tt.want {
				t.Errorf("Turn.Preview() = %q, want %q", got, tt.want)
			}
		})
	}
}
