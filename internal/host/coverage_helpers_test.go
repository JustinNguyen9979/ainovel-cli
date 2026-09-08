package host

import (
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/arbiter"
	"github.com/JustinNguyen9979/ainovel-cli/internal/utils"
)

func TestHostCoverageHelpers(t *testing.T) {
	if (arbiter.InterventionFacts{}).QueueHead() != 0 || (arbiter.InterventionFacts{PendingRewrites: []int{4}}).QueueHead() != 4 {
		t.Fatal("queue head mismatch")
	}
	if streamHeaderFallback("read_chapter") != "✻ read_chapter" {
		t.Fatal("stream fallback header mismatch")
	}
	if got := localizedStoryWarning(utils.LanguageVI, "scope", errTest("bad")); got == "" {
		t.Fatal("localized story warning empty")
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestHostOptionAppliesFields(t *testing.T) {
	var opts newOptions
	WithFileLog("test.log", true)(&opts)
	if opts.logFile != "test.log" || !opts.logAlsoStderr {
		t.Fatalf("options = %+v", opts)
	}
}

func TestHostLanguageHelpers(t *testing.T) {
	if localizedStoryWarning(utils.LanguageVI, "scope", errTest("bad")) == localizedStoryWarning(utils.LanguageZH, "scope", errTest("bad")) {
		t.Fatal("languages should differ")
	}
}
