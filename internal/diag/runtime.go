package diag

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/JustinNguyen9979/ainovel-cli/internal/domain"
	"github.com/JustinNguyen9979/ainovel-cli/internal/store"
	"github.com/voocel/agentcore"
)

const (
	logTailCap   = 200 << 10 // log chỉ lấy phần đuôi 200KB (vòng lặp là hiện tượng xảy ra ở phần gần nhất)
	sessionTail  = 80        // số sự kiện đuôi khung xương (xem thứ tự dispatch)
	repeatWindow = 150       // tổng hợp lặp chỉ xét nhiêu sự kiện gần nhất —— trong chạy dài, công cụ tích lũy hàng trăm lần là bình thường,
	// vòng lặp thực sự là tập trung cao ở phần gần nhất; dùng cửa sổ thay vì tổng tích lũy, tránh nhầm "tiến triển bình thường" thành "vòng lặp chết".
	recentAgents = 2  // số phiên agent phụ hoạt động gần nhất được quét thêm
	repeatMin    = 3  // lặp bao nhiêu lần mới được coi là "tín hiệu tần suất cao"
	repeatTopN   = 12 // tối đa liệt kê bao nhiêu chữ ký lặp
)

// RuntimeCapture là kết quả thu thập thời gian chạy đã được ẩn danh. Chỉ chứa tín hiệu thời gian chạy;
// phase/flow/chương và trạng thái sáng tác do Report.Stats mang, không lặp lại ở đây.
type RuntimeCapture struct {
	GoOS, GoArch  string
	Models        []RoleModel  // provider/model thực tế có hiệu lực của mỗi phiên (thu thập từ _meta)
	CurrentStep   string       // điểm khôi phục mới nhất: scope.step
	StuckStep     string       // step liên tiếp giống nhau ở đuôi; "" = không bị kẹt
	StuckCount    int          // số lần liên tiếp
	Repeats       []RepeatStat // chữ ký lặp top-N (tín hiệu vòng lặp)
	DupContent    []DupStat    // cùng sha xuất hiện nhiều lần (tái tạo cùng đoạn văn)
	LogKinds      map[string]int
	LogErrors     int
	LogWarns      int
	StopGuard     int
	Tail          []SkelEvent // N sự kiện khung xương cuối (xem thứ tự)
	RedactedTexts int         // tổng số khối văn bản đã mã hóa (tự kiểm tra ẩn danh)
	Sources       []string    // các nguồn thực tế đã đọc (tự kiểm tra)
}

// RoleModel ghi lại provider/model thực tế được dùng trong một phiên.
type RoleModel struct {
	Agent, Provider, Model string
}

// RepeatStat là một chữ ký lặp và số lần xuất hiện.
type RepeatStat struct {
	Sig   string
	Count int
}

// DupStat là số lần cùng một đoạn văn bản đã ẩn danh xuất hiện lặp lại.
type DupStat struct {
	Sha   string
	Count int
}

// sessionLine phân tích một dòng trong sessions/*.jsonl: nhúng agentcore.Message + _meta tùy chọn.
type sessionLine struct {
	agentcore.Message
	Meta *struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	} `json:"_meta"`
}

var kindRe = regexp.MustCompile(`kind=(\S+)`)

// CaptureRuntime chỉ đọc từ thư mục output để thu thập tín hiệu thời gian chạy và tổng hợp sau khi ẩn danh.
// Mọi nguồn bị thiếu đều được xử lý an toàn theo cơ chế giảm cấp (không báo lỗi), cố gắng hết sức.
func CaptureRuntime(s *store.Store) RuntimeCapture {
	rc := RuntimeCapture{GoOS: runtime.GOOS, GoArch: runtime.GOARCH, LogKinds: map[string]int{}}

	rc.CurrentStep, rc.StuckStep, rc.StuckCount = analyzeCheckpoints(s.Checkpoints.All())
	captureSessions(s.Dir(), &rc)
	captureLog(s.Dir(), &rc)
	return rc
}

// analyzeCheckpoints lấy step mới nhất và tính số lần step cuối liên tiếp giống nhau (tín hiệu bị kẹt).
func analyzeCheckpoints(cps []domain.Checkpoint) (current, stuck string, count int) {
	if len(cps) == 0 {
		return "", "", 0
	}
	key := func(c domain.Checkpoint) string { return fmt.Sprintf("%s.%s", c.Scope, c.Step) }
	current = key(cps[len(cps)-1])
	n := 1
	for i := len(cps) - 2; i >= 0; i-- {
		if key(cps[i]) == current {
			n++
		} else {
			break
		}
	}
	if n >= repeatMin {
		stuck, count = current, n
	}
	return current, stuck, count
}

// captureSessions quét các phiên Worker hoạt động gần nhất và tổng hợp sau khi ẩn danh.
func captureSessions(dir string, rc *RuntimeCapture) {
	sessDir := filepath.Join(dir, "meta", "sessions")
	files := sessionFiles(sessDir)

	repeats := map[string]int{}
	dups := map[string]int{}
	models := map[string]RoleModel{}

	for _, f := range files {
		evs := scanSession(filepath.Join(sessDir, f.path), f.agent, rc, models)
		// Tổng hợp chỉ xét cửa sổ gần nhất: trong chạy dài, subagent/novel_context tích lũy hàng trăm lần là tiến triển bình thường,
		// không phải vòng lặp; vòng lặp chết thực sự là tập trung cao ở phần gần nhất.
		aggregateRepeats(f.agent, tailEvents(evs, repeatWindow), repeats, dups)
		// Các file được xếp theo thời gian hoạt động giảm dần; lấy phiên không rỗng đầu tiên.
		if len(rc.Tail) == 0 && len(evs) > 0 {
			rc.Tail = tailEvents(evs, sessionTail)
		}
		rc.Sources = append(rc.Sources, "sessions/"+f.path)
	}
	rc.Repeats = topRepeats(repeats)
	rc.DupContent = topDups(dups)
	rc.Models = sortedModels(models)
}

type sessionFile struct {
	path  string // tương đối so với sessDir
	agent string
}

// sessionFiles trả về các phiên Worker hoạt động gần nhất.
func sessionFiles(sessDir string) []sessionFile {
	agentsDir := filepath.Join(sessDir, "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil
	}
	type withTime struct {
		name string
		mod  int64
	}
	var agents []withTime
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		if info, err := e.Info(); err == nil {
			agents = append(agents, withTime{e.Name(), info.ModTime().UnixNano()})
		}
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].mod > agents[j].mod })
	out := make([]sessionFile, 0, min(len(agents), recentAgents))
	for i, a := range agents {
		if i >= recentAgents {
			break
		}
		stem := strings.TrimSuffix(a.name, ".jsonl")
		out = append(out, sessionFile{path: filepath.Join("agents", a.name), agent: stem})
	}
	return out
}

// scanSession đọc một file phiên, ẩn danh từng dòng, thu thập chuỗi sự kiện và model theo từng agent.
// Tổng hợp lặp/đoạn trùng không thực hiện ở đây —— giao cho aggregateRepeats tính trên cửa sổ gần nhất.
func scanSession(path, agent string, rc *RuntimeCapture, models map[string]RoleModel) []SkelEvent {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var evs []SkelEvent
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 8<<20)
	for sc.Scan() {
		var sl sessionLine
		if json.Unmarshal(sc.Bytes(), &sl) != nil {
			continue
		}
		ev := redactMessage(agent, sl.Message)
		evs = append(evs, ev)
		rc.RedactedTexts += ev.Redacted
		if sl.Meta != nil && (sl.Meta.Provider != "" || sl.Meta.Model != "") {
			models[agent] = RoleModel{Agent: agent, Provider: sl.Meta.Provider, Model: sl.Meta.Model}
		}
	}
	return evs
}

// aggregateRepeats tích lũy chữ ký lặp và đoạn văn trùng trên cửa sổ sự kiện đã cho.
func aggregateRepeats(agent string, evs []SkelEvent, repeats, dups map[string]int) {
	for _, ev := range evs {
		for _, t := range ev.Tools {
			sig := agent + " · " + t.Name
			if t.Invalid {
				sig += " (args invalid)"
			}
			repeats[sig]++
		}
		if ev.ErrClass != "" {
			repeats[agent+" · err: "+ev.ErrClass]++
		}
		if ev.TextSha != "" {
			dups[ev.TextSha]++
		}
	}
}

func tailEvents(evs []SkelEvent, n int) []SkelEvent {
	if len(evs) <= n {
		return evs
	}
	return evs[len(evs)-n:]
}

// captureLog đọc phần đuôi log, chỉ tổng hợp tín hiệu cấu trúc (kind/error/warn/stop_guard),
// không đưa dòng log thô vào gói —— Detail có thể chứa nội dung bản thảo.
func captureLog(dir string, rc *RuntimeCapture) {
	path := filepath.Join(dir, "logs", "tui.log")
	tail, ok := readTail(path)
	if !ok {
		path = filepath.Join(dir, "logs", "headless.log")
		tail, ok = readTail(path)
	}
	if !ok {
		return
	}
	rc.Sources = append(rc.Sources, "logs/"+filepath.Base(path)+" (đuôi)")

	sc := bufio.NewScanner(bytes.NewReader(tail))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.Contains(line, "level=ERROR"):
			rc.LogErrors++
		case strings.Contains(line, "level=WARN"):
			rc.LogWarns++
		}
		if m := kindRe.FindStringSubmatch(line); m != nil {
			rc.LogKinds[m[1]]++
		}
		if strings.Contains(line, "stop_guard") {
			rc.StopGuard++
		}
	}
}

// readTail đọc logTailCap byte ở đuôi file và bỏ nửa dòng đầu có thể bị cắt.
func readTail(path string) ([]byte, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, false
	}
	size := info.Size()
	var off int64
	if size > logTailCap {
		off = size - logTailCap
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return nil, false
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, false
	}
	if off > 0 {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		}
	}
	return data, true
}

func topRepeats(m map[string]int) []RepeatStat {
	var out []RepeatStat
	for sig, c := range m {
		if c >= repeatMin {
			out = append(out, RepeatStat{Sig: sig, Count: c})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Sig < out[j].Sig
	})
	if len(out) > repeatTopN {
		out = out[:repeatTopN]
	}
	return out
}

func topDups(m map[string]int) []DupStat {
	var out []DupStat
	for sha, c := range m {
		if c >= repeatMin {
			out = append(out, DupStat{Sha: sha, Count: c})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Sha < out[j].Sha
	})
	return out
}

func sortedModels(m map[string]RoleModel) []RoleModel {
	out := make([]RoleModel, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Agent < out[j].Agent })
	return out
}
