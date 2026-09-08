package host

import (
	"strings"
	"testing"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	storepkg "github.com/JustinNguyen9979/ainovel-cli/internal/store"
)

func TestDescribeResumeStateLabels(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		p    *domain.Progress
		want string
	}{
		{"premise", &domain.Progress{Phase: domain.PhasePremise}, "规划阶段"},
		{"outline", &domain.Progress{Phase: domain.PhaseOutline}, "规划阶段"},
		{"reviewing", &domain.Progress{Phase: domain.PhaseWriting, Flow: domain.FlowReviewing}, "审阅中断"},
		{"in progress", &domain.Progress{Phase: domain.PhaseWriting, InProgressChapter: 4}, "第 4 章进行中"},
		{"next", &domain.Progress{Phase: domain.PhaseWriting, TotalChapters: 10, CompletedChapters: []int{1, 2}}, "从第 3 章继续"},
		{"other", &domain.Progress{Phase: domain.PhaseInit}, "恢复"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := describeResume(st, tc.p)
			if err != nil || !strings.Contains(got, tc.want) {
				t.Fatalf("label=%q err=%v want %q", got, err, tc.want)
			}
		})
	}
	p := &domain.Progress{Phase: domain.PhaseWriting, Flow: domain.FlowRewriting, PendingRewrites: []int{2, 3}}
	if got, err := describeResume(st, p); err != nil || !strings.Contains(got, "重写") {
		t.Fatalf("rewrite label=%q err=%v", got, err)
	}
	p.Flow = domain.FlowPolishing
	if got, err := describeResume(st, p); err != nil || !strings.Contains(got, "打磨") {
		t.Fatalf("polish label=%q err=%v", got, err)
	}
	if label, err := resumeLabel(st); err != nil || label != "" {
		t.Fatalf("empty resume label=%q err=%v", label, err)
	}
}

func TestDescribeResumePendingCommitAndArcLabels(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	if err := st.Signals.SavePendingCommit(domain.PendingCommit{Chapter: 3}); err != nil {
		t.Fatal(err)
	}
	p := &domain.Progress{Phase: domain.PhaseWriting}
	if got, err := describeResume(st, p); err != nil || !strings.Contains(got, "第 3 章提交中断") {
		t.Fatalf("pending commit label=%q err=%v", got, err)
	}
	if err := st.Signals.ClearPendingCommit(); err != nil {
		t.Fatal(err)
	}
	if got, err := describeArcEndLabel(st, &domain.Progress{}); err != nil || got != "" {
		t.Fatalf("non-layered arc label=%q err=%v", got, err)
	}
}

func TestDescribeArcEndLabels(t *testing.T) {
	st := storepkg.NewStore(t.TempDir())
	if err := st.Init(); err != nil {
		t.Fatal(err)
	}
	volumes := []domain.VolumeOutline{{Index: 1, Title: "第一卷", Arcs: []domain.ArcOutline{
		{Index: 1, Title: "第一弧", Chapters: []domain.OutlineEntry{{Title: "一"}}},
		{Index: 2, Title: "第二弧", EstimatedChapters: 3},
	}}}
	if err := st.Outline.SaveLayeredOutline(volumes); err != nil {
		t.Fatal(err)
	}
	p := &domain.Progress{Phase: domain.PhaseWriting, Layered: true, CompletedChapters: []int{1}}
	if got, err := describeArcEndLabel(st, p); err != nil || !strings.Contains(got, "弧末评审待处理") {
		t.Fatalf("review label=%q err=%v", got, err)
	}
	if err := st.World.SaveReview(domain.ReviewEntry{Chapter: 1, Scope: "arc", Verdict: "accept"}); err != nil {
		t.Fatal(err)
	}
	if got, err := describeArcEndLabel(st, p); err != nil || !strings.Contains(got, "弧摘要待生成") {
		t.Fatalf("arc summary label=%q err=%v", got, err)
	}
	if err := st.Summaries.SaveArcSummary(domain.ArcSummary{Volume: 1, Arc: 1}); err != nil {
		t.Fatal(err)
	}
	if got, err := describeArcEndLabel(st, p); err != nil || !strings.Contains(got, "待展开下一弧") {
		t.Fatalf("expansion label=%q err=%v", got, err)
	}

	volumes[0].Arcs = []domain.ArcOutline{{Index: 1, Chapters: []domain.OutlineEntry{{Title: "一"}}}}
	if err := st.Outline.SaveLayeredOutline(volumes); err != nil {
		t.Fatal(err)
	}
	if err := st.Summaries.SaveVolumeSummary(domain.VolumeSummary{Volume: 1}); err != nil {
		t.Fatal(err)
	}
	if got, err := describeArcEndLabel(st, p); err != nil || !strings.Contains(got, "待决策下一卷") {
		t.Fatalf("new volume label=%q err=%v", got, err)
	}
}
