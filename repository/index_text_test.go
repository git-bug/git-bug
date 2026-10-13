package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTextTokens(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{"empty", "", nil},
		{"only spaces and punctuation", "  -- ,. ", nil},
		{"lowercased and stemmed", "Crashes CRASHED crashing", []string{"crash", "crash", "crash"}},
		{"stop words dropped", "the server is on fire", []string{"server", "fire"}},
		{"only stop words", "the", nil},
		{"split on punctuation", "foo-bar file.go a/b", []string{"foo", "bar", "file", "go", "b"}},
		{"possessive dropped", "the server's log", []string{"server", "log"}},
		{"apostrophe inside a word", "don’t", []string{"dont"}},
		{"quotes around a word", "'quoted'", []string{"quot"}},
		{"digits kept", "v2 error 404", []string{"v2", "error", "404"}},
		{"diacritics kept", "café crème", []string{"café", "crème"}},
		{"other scripts kept", "Ошибка сервера", []string{"ошибка", "сервера"}},
		{"CJK in pairs", "数据库连接", []string{"数据", "据库", "库连", "连接"}},
		{"CJK single character", "库", []string{"库"}},
		{"CJK runs split by punctuation", "数据，库", []string{"数据", "库"}},
		{"CJK and latin mixed", "数据库v2连接", []string{"数据", "据库", "v2", "连接"}},
		{"Japanese kana", "データベース", []string{"デー", "ータ", "タベ", "ベー", "ース"}},
		{"Korean", "데이터 베이스", []string{"데이", "이터", "베이", "이스"}},
		{"query syntax is plain text", `-x +y title:z (abc "server* NEAR`, []string{"x", "y", "titl", "z", "abc", "server", "near"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, textTokens(tt.text))
		})
	}
}
