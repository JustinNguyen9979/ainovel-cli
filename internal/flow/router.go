// Package flow triển khai Flow Router theo ngành dọc: Host quyết định dựa trên thực tế
// xem SubAgent nào sẽ được gọi tiếp theo và làm gì.
//
// Nguyên tắc thiết kế:
//   - Route là hàm thuần túy: đầu vào là State, đầu ra là *Instruction. Không có IO, không gọi Store, có thể unit test độc lập.
//   - State được LoadState (không thuần túy) xây dựng từ Store, đọc toàn bộ dữ liệu cần thiết cho routing một lần.
//   - Trả về nil là hợp lệ: chưa có chỉ thị Worker nào suy ra được từ dữ liệu xác định;
//     Engine sẽ xử lý theo trạng thái kết thúc, phán quyết khởi động hoặc chờ người dùng.
//
// Router bao gồm các quyết định kiểu "tra bảng" (bước tiếp theo mỗi chương, hậu xử lý cuối cung truyện, điều phối theo hàng đợi),
// không bao gồm các quyết định kiểu "hiểu ngữ nghĩa" (chọn kiến trúc sư, xử lý Steer của người dùng, xuất tóm tắt).
package flow

import (
	"fmt"
	"strings"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	storepkg "github.com/JustinNguyen9979/ainovel-cli/internal/store"
)

// plannerForTier suy ra planner từ cấp lập kế hoạch đã lưu.
func plannerForTier(tier domain.PlanningTier) string {
	if tier == domain.PlanningTierShort {
		return "architect_short"
	}
	return "architect_long"
}

// Instruction chỉ thị Worker và nhiệm vụ mà Engine sẽ chạy trực tiếp ở bước tiếp theo.
type Instruction struct {
	Agent   string // architect_long / architect_short / writer / editor
	Task    string // mô tả nhiệm vụ giao cho SubAgent
	Reason  string // lý do định tuyến, dùng cho sự kiện, log và phán quyết lỗi
	Chapter int    // số chương liên quan đến nhiệm vụ writer (tiếp tục/viết lại/đánh bóng); 0 = không liên quan (nhiệm vụ editor/architect)
}

// State là đầu vào của Route: tất cả dữ liệu thực tế phải được khai báo rõ ràng ở đây, Route không được đọc Store nội bộ.
type State struct {
	Progress *domain.Progress

	// Chương đã hoàn thành cuối cùng (phần tử cuối của Progress.CompletedChapters); 0 nghĩa là chưa bắt đầu viết.
	LastCompleted int

	// Thông tin ranh giới cung truyện của chương trước; khi IsArcEnd=false các trường còn lại không có ý nghĩa.
	// Nên là nil khi LastCompleted=0 hoặc không ở chế độ Layered.
	ArcBoundary *storepkg.ArcBoundary

	// Ba dữ liệu hậu xử lý cuối cung truyện: đánh giá / tóm tắt cung / tóm tắt tập đã hoàn thành chưa.
	HasArcReview     bool
	HasArcSummary    bool
	HasVolumeSummary bool

	// Các mục thiếu trong cài đặt nền tảng (tín hiệu bổ sung trong giai đoạn lập kế hoạch).
	FoundationMissing []string

	PlanningTier           domain.PlanningTier
	HasGlobalReview        bool
	ImmediateFeedbackCount int
	AggregateRefresh       *AggregateRefresh
}

type AggregateKind string

const (
	AggregateArcReview     AggregateKind = "arc_review"
	AggregateArcSummary    AggregateKind = "arc_summary"
	AggregateVolumeSummary AggregateKind = "volume_summary"
	AggregateGlobalReview  AggregateKind = "global_review"
)

type AggregateRefresh struct {
	Kind                                  AggregateKind
	Volume, Arc, StartChapter, EndChapter int
}

// Route trả về chỉ thị xác định dựa trên dữ liệu thực tế; khi nil, Engine xử lý theo ngữ cảnh gọi.
//
// Mức độ ưu tiên quyết định (loại trừ lẫn nhau, khớp mục đầu tiên từ trên xuống):
//  1. Phase=Complete        → nil (LLM xuất tóm tắt)
//  2. Phase!=Writing        → nil (LLM chọn kiến trúc sư / bổ sung kế hoạch)
//  3. PendingRewrites không rỗng → writer viết lại/chỉnh sửa theo hàng đợi
//  4. Flow=Reviewing        → nil (nhánh ngủ: hiện không có bên nào ghi trạng thái này; lúc đánh giá Flow thực tế là writing)
//  5. Flow=Steering         → nil (đang xử lý can thiệp người dùng)
//  6. Thiếu đánh giá cuối mạch truyện → editor(arc review)
//  7. Có đánh giá nhưng thiếu tóm tắt mạch truyện → editor(arc summary)
//  8. Cuối tập có tóm tắt mạch truyện nhưng thiếu tóm tắt tập → editor(volume summary)
//  9. Mạch truyện tiếp theo là khung → architect_long(expand_arc)
//
// 10. Cuối tập cần quyết định tập tiếp theo       → architect_long(append_volume / complete_book)
// 11. Các trường hợp còn lại                  → writer(viết next_chapter)
func Route(s State) *Instruction {
	p := s.Progress
	if p == nil {
		return nil
	}

	// 1. Trạng thái kết thúc: Host tạo tóm tắt xác định từ dữ liệu Store.
	if p.Phase == domain.PhaseComplete {
		return nil
	}

	// 2. Tự giao cùng planner bổ sung thiết lập nếu danh tính planner đã được lưu.
	if p.Phase != domain.PhaseWriting {
		if len(s.FoundationMissing) > 0 && s.PlanningTier != "" {
			task := fmt.Sprintf("Bổ sung các mục thiết lập nền còn thiếu: %s; mục book dùng save_book, các thiết lập nền khác dùng save_foundation để lưu đúng type", strings.Join(s.FoundationMissing, ", "))
			if len(s.FoundationMissing) == 1 && s.FoundationMissing[0] == "foundation_audit" {
				task = "Thiết lập nền đã đủ: gọi novel_context để đọc toàn bộ artifact và foundation_status.fingerprint, kiểm tra tính nhất quán giữa các file rồi gọi audit_foundation; nếu có vấn đề thì sửa trước và kiểm tra lại"
			}
			return &Instruction{
				Agent:  plannerForTier(s.PlanningTier),
				Task:   task,
				Reason: "Thiết lập nền chưa đủ; tiếp tục giao cùng planner theo danh sách thiếu",
			}
		}
		return nil
	}

	// 3. Hàng đợi viết lại/đánh bóng được ưu tiên (dữ liệu đã được tầng công cụ ghi đĩa, Router chỉ điều phối theo danh sách)
	if len(p.PendingRewrites) > 0 {
		ch := p.PendingRewrites[0]
		verb := "Viết lại"
		if p.Flow == domain.FlowPolishing {
			verb = "Đánh bóng"
		}
		return &Instruction{
			Agent:   "writer",
			Task:    fmt.Sprintf("%s chương %d", verb, ch),
			Reason:  fmt.Sprintf("Hàng đợi PendingRewrites còn %d chương", len(p.PendingRewrites)),
			Chapter: ch,
		}
	}

	// 4. Đang đánh giá → trả quyền cho LLM. Đây hiện là nhánh ngủ: save_review chỉ đặt Flow thành
	// writing/rewriting/polishing, không có đường dẫn sản xuất nào đặt reviewing. Giữ nhánh này đối xứng
	// với Steering và để router nhường quyền cho LLM nếu sau này giai đoạn đánh giá đặt reviewing rõ ràng.
	if p.Flow == domain.FlowReviewing {
		return nil
	}

	// 5. Đang xử lý can thiệp của người dùng: Arbiter đang phán quyết, Engine không chiếm quyền.
	if p.Flow == domain.FlowSteering {
		return nil
	}

	// Phục hồi các tổng hợp bị vô hiệu hóa sau sửa đổi ngoài: tạo lại trước khi lập kế hoạch tiếp.
	if refresh := s.AggregateRefresh; refresh != nil {
		switch refresh.Kind {
		case AggregateArcReview:
			return &Instruction{
				Agent: "editor",
				Task: fmt.Sprintf(
					"Đánh giá cung %d tập %d (chương %d-%d): gọi novel_context(chapter=%d), save_review với scope=arc, chapter=%d",
					refresh.Arc, refresh.Volume, refresh.StartChapter, refresh.EndChapter, refresh.EndChapter, refresh.EndChapter,
				),
				Reason: "Thiếu đánh giá cung truyện sau khi cập nhật",
			}
		case AggregateArcSummary:
			return &Instruction{
				Agent:  "editor",
				Task:   fmt.Sprintf("Tạo tóm tắt cung %d tập %d, ảnh chụp nhân vật và quy tắc sáng tác (save_arc_summary)", refresh.Arc, refresh.Volume),
				Reason: "Thiếu tóm tắt cung truyện sau khi cập nhật",
			}
		case AggregateVolumeSummary:
			return &Instruction{
				Agent:  "editor",
				Task:   fmt.Sprintf("Tạo tóm tắt tập %d (save_volume_summary)", refresh.Volume),
				Reason: "Thiếu tóm tắt tập sau khi cập nhật",
			}
		case AggregateGlobalReview:
			return &Instruction{
				Agent:  "editor",
				Task:   fmt.Sprintf("Đánh giá %d chương đầu: gọi novel_context(chapter=%d), save_review với scope=global, chapter=%d", refresh.EndChapter, refresh.EndChapter, refresh.EndChapter),
				Reason: "Thiếu đánh giá toàn cục sau khi cập nhật",
			}
		}
	}

	if s.ImmediateFeedbackCount > 0 {
		return &Instruction{
			Agent:  plannerForTier(s.PlanningTier),
			Task:   "Chỉ xử lý writer_feedback từ novel_context: đối chiếu diễn biến đã xảy ra với kế hoạch tiếp theo; nếu cần thì gọi revise_outline hoặc công cụ cấu trúc tương ứng, nếu không cần thì gọi resolve_outline_feedback; không xử lý foundation_status hay quy hoạch khác, sau khi lưu hãy kết thúc bằng một câu",
			Reason: fmt.Sprintf("Có %d phản hồi sửa đổi bên ngoài chưa được truyền vào kế hoạch tiếp theo", s.ImmediateFeedbackCount),
		}
	}

	// Hậu xử lý cuối cung truyện trong chế độ phân lớp
	if p.Layered && s.ArcBoundary != nil && s.ArcBoundary.IsArcEnd {
		b := s.ArcBoundary
		switch {
		case !s.HasArcReview:
			return &Instruction{
				Agent: "editor",
				Task: fmt.Sprintf(
					"Đánh giá cung %d tập %d (chương %d-%d): gọi novel_context(chapter=%d), save_review với scope=arc, chapter=%d; issues[].chapters chỉ được nằm trong khoảng này",
					b.Arc, b.Volume, b.StartChapter, b.EndChapter, b.EndChapter, b.EndChapter,
				),
				Reason: "Đánh giá cuối cung truyện chưa hoàn thành",
			}
		case !s.HasArcSummary:
			return &Instruction{
				Agent:  "editor",
				Task:   fmt.Sprintf("Tạo tóm tắt cung %d tập %d (save_arc_summary)", b.Arc, b.Volume),
				Reason: "Tóm tắt cung truyện chưa hoàn thành",
			}
		case b.IsVolumeEnd && !s.HasVolumeSummary:
			return &Instruction{
				Agent:  "editor",
				Task:   fmt.Sprintf("Tạo tóm tắt tập %d (save_volume_summary)", b.Volume),
				Reason: "Tóm tắt tập chưa hoàn thành",
			}
		case b.NeedsExpansion && b.NextArc > 0:
			return &Instruction{
				Agent:  "architect_long",
				Task:   fmt.Sprintf("Mở rộng cung %d tập %d (save_foundation type=expand_arc)", b.NextArc, b.NextVolume),
				Reason: "Skeleton cung truyện tiếp theo cần được mở rộng",
			}
		case b.NeedsNewVolume:
			return &Instruction{
				Agent:  "architect_long",
				Task:   "Tạo tập tiếp theo: đánh giá theo danh sách hoàn kết rồi gọi save_foundation. Truyện tiếp tục → type=append_volume; gần kết thúc → type=append_volume với \"final\": true; đã thỏa mọi điều kiện kết thúc → type=complete_book. Cả ba lựa chọn đều phải kèm tham số reason nêu lý do phán quyết",
				Reason: "Cuối tập cần quyết định thêm tập mới, tập kết hay kết thúc toàn bộ tác phẩm",
			}
		}
	}

	// Sách không phân lớp được đánh giá global theo ReviewInterval.
	if !p.Layered && s.LastCompleted > 0 {
		if due, reason := domain.ShouldReview(len(p.CompletedChapters)); due && !s.HasGlobalReview {
			return &Instruction{
				Agent:  "editor",
				Task:   fmt.Sprintf("Đánh giá toàn cục %d chương đầu (save_review scope=global, chapter=%d)", s.LastCompleted, s.LastCompleted),
				Reason: reason,
			}
		}
	}

	// Đại cương không phân lớp đã hết: không được phát chương vượt quá giới hạn.
	next := p.NextChapter()
	if next <= 0 {
		return nil
	}
	if !p.Layered && p.TotalChapters > 0 && next > p.TotalChapters {
		return &Instruction{
			Agent: plannerForTier(s.PlanningTier),
			Task: fmt.Sprintf(
				"Đại cương không phân lớp đã hoàn tất (đã xong %d chương trên tổng %d): nếu câu chuyện đã khép lại, gọi save_foundation(type=complete_book); nếu cần tiếp tục, dùng revise_outline để bổ sung kế hoạch từ chương %d",
				len(p.CompletedChapters), p.TotalChapters, next,
			),
			Reason: "Đại cương không phân lớp đã hết, cần quyết định hoàn tất hoặc tiếp tục",
		}
	}

	// Tiếp tục viết bình thường.
	return &Instruction{
		Agent:   "writer",
		Task:    fmt.Sprintf("Viết chương %d", next),
		Reason:  "Tiếp tục viết chương tiếp theo",
		Chapter: next,
	}
}
