package utils

import "testing"

func TestDecodeTextHandlesBOMAndGB18030(t *testing.T) {
	if got := DecodeText([]byte("\xef\xbb\xbfhello")); got != "hello" {
		t.Fatalf("BOM decode = %q", got)
	}
	if got := DecodeText([]byte{0xc4, 0xe3, 0xba, 0xc3}); got != "你好" {
		t.Fatalf("GB18030 decode = %q", got)
	}
}
