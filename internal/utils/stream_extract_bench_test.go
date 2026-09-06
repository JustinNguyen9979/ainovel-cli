package utils

import "testing"

func BenchmarkJSONFieldExtractor(b *testing.B) {
	const input = `{"name":"draft_chapter","arguments":{"content":"Một chương truyện dài với ký tự \\"đặc biệt\\"."}}`
	for b.Loop() {
		ext := NewFieldExtractor("content")
		ext.Feed(input)
	}
}

func BenchmarkStreamFilter(b *testing.B) {
	const input = `prefix {"content":"Một đoạn văn bản"} suffix`
	for b.Loop() {
		filter := NewStreamFilter("content")
		filter.Feed(input)
	}
}
