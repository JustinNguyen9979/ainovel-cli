package notify

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNotifierFilteringAndCommandDelivery(t *testing.T) {
	if (*Notifier)(nil).allows("x") {
		t.Fatal("nil notifier should reject")
	}
	n := New("", []string{KindBudget})
	if !n.allows(KindBudget) || n.allows(KindRunEnd) {
		t.Fatal("event filtering mismatch")
	}
	if New("", nil).allows("anything") != true {
		t.Fatal("nil event filter should allow all")
	}
	nt := Notification{Kind: KindBudget, Level: "warn", Title: "title", Body: "body"}
	cmd := New("exit 0", nil)
	if err := cmd.deliverError(nt); err != nil {
		t.Fatal(err)
	}
	if err := New("exit 3", nil).deliverError(nt); err == nil {
		t.Fatal("failed notification command should return error")
	}
	if got := notificationEnv(nt); len(got) == 0 || !containsEnv(got, "NOTIFY_KIND="+KindBudget) || !containsEnv(got, "NOTIFY_BODY=body") {
		t.Fatal("notification env missing fields")
	}
	if err := runCommand(context.Background(), "exit 0", nt); err != nil {
		t.Fatal(err)
	}
	if got := appleScriptString(`a\"b`); !strings.Contains(got, `\\`) || !strings.Contains(got, `\"`) {
		t.Fatalf("AppleScript escaping = %q", got)
	}
	n.deliver(nt)
	_ = time.Second
}

func containsEnv(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}
