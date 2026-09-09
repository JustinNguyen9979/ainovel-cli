# ainovel-cli-vn

CLI sáng tác tiểu thuyết dài kỳ bằng AI, với giao diện TUI tiếng Việt và hỗ trợ viết bằng tiếng Việt hoặc tiếng Trung.

## Tính năng

- Điều phối nhiều agent: Architect, Writer, Editor và Arbiter.
- Lập kế hoạch cuộn cho tiểu thuyết dài kỳ.
- Checkpoint và khôi phục theo từng bước.
- Can thiệp thời gian thực trong lúc sáng tác.
- Đánh giá tính nhất quán, nhân vật, nhịp truyện và văn phong.
- Hỗ trợ OpenRouter, Anthropic, Gemini, OpenAI, DeepSeek, Ollama và proxy tương thích.
- Tích hợp `ainovel-cli-vn` qua npm cho macOS, Linux và Windows trên x64/arm64.

## Yêu cầu hệ thống

| Cách dùng | Yêu cầu |
|---|---|
| npm | Node.js >= 18; macOS/Linux/Windows x64 hoặc arm64 |
| Build source | Go 1.25.5 trở lên |
| Sáng tác | API key của provider hoặc Ollama local |

## Cài đặt bằng npm

Cài launcher toàn cục:

```bash
npm install --global ainovel-cli-vn@latest
ainovel-cli-vn
```

Hoặc chạy trực tiếp bằng `npx`:

```bash
npx ainovel-cli-vn@latest
```

Launcher npm không chứa binary Go. Ở lần chạy đầu, launcher tải archive native đúng với hệ điều hành và CPU từ GitHub Release tương ứng, kiểm tra SHA-256 rồi cache bên ngoài `node_modules`. Vì vậy npm install cần internet ở lần chạy đầu tiên; các lần sau có thể chạy từ cache đã xác minh. Dùng `@latest` để luôn cài và chạy bản phát hành mới nhất.

Package hỗ trợ các target sau:

- Linux x64 và arm64.
- macOS Intel và Apple Silicon.
- Windows x64 và arm64.

Các hệ điều hành hoặc kiến trúc khác sẽ được báo là không được hỗ trợ thay vì chạy binary không phù hợp.

## Cấu hình

Lần chạy đầu tiên trong terminal tương tác sẽ mở Setup Wizard. Cấu hình được lưu tại:

```text
~/.ainovel/config.json
```

Có thể xem mẫu tại [`config.example.jsonc`](config.example.jsonc). Không commit file cấu hình thật vì file có thể chứa API key.

Ví dụ dùng Ollama local:

```json
{
  "language": "vi",
  "provider": "ollama",
  "model": "qwen3:14b",
  "providers": {
    "ollama": {
      "base_url": "http://localhost:11434/v1",
      "models": ["qwen3:14b"]
    }
  },
  "style": "default"
}
```

Ví dụ dùng OpenRouter:

```json
{
  "language": "vi",
  "provider": "openrouter",
  "model": "google/gemini-2.5-flash",
  "providers": {
    "openrouter": {
      "api_key": "sk-or-v1-...",
      "base_url": "https://openrouter.ai/api/v1"
    }
  }
}
```

`language` nhận `vi` hoặc `zh`. `reasoning_effort` nhận `off`, `low`, `medium`, `high`, `xhigh` hoặc `max` tùy khả năng model.

## Bắt đầu sáng tác

Sau khi chạy `ainovel-cli-vn`, nhập yêu cầu vào TUI rồi nhấn Enter. Ví dụ:

```text
Tiểu thuyết fantasy Việt Nam dài 80 chương, nhân vật chính là cô gái nghèo bị bán vào phủ làm tỳ nữ. Kết thúc có hậu.
```

Hệ thống sẽ lập kế hoạch, viết từng chương, chạy kiểm tra và lưu tiến độ. Nếu bị mất mạng hoặc tắt chương trình, chạy lại trong cùng thư mục để khôi phục từ checkpoint.

Chế độ headless yêu cầu cấu hình đã tồn tại:

```bash
ainovel-cli-vn --headless --prompt "Viết một truyện trinh thám ngắn ở Hà Nội"
```

## Lệnh TUI

| Lệnh | Mô tả |
|---|---|
| `/help` | Xem trợ giúp và phím tắt |
| `/model` | Chuyển provider/model hoặc cấu hình vai trò |
| `/diag` | Chạy chẩn đoán tiến độ và tính nhất quán |
| `/import <đường-dẫn>` | Nhập một tiểu thuyết có sẵn |
| `/simulate` | Tạo hồ sơ văn phong từ văn mẫu |
| `/export` | Xuất truyện ra TXT |
| `/export <đường-dẫn>.epub` | Xuất truyện ra EPUB |
| `/cocreate` | Lập kế hoạch cho giai đoạn tiếp theo |

| Phím | Chức năng |
|---|---|
| `Enter` | Gửi yêu cầu hoặc can thiệp |
| `Tab` | Chuyển chế độ khởi động |
| `Esc` | Xóa input hoặc đóng panel |
| `Ctrl+C` | Dừng an toàn và thoát |

## Quy tắc và văn phong

Tạo file Markdown trong một trong các thư mục sau:

```text
~/.ainovel/rules/       # Quy tắc dùng chung
./.ainovel/rules/       # Quy tắc riêng cho thư mục dự án
```

Quy tắc được chuẩn hóa khi khởi động. Các kiểm tra cơ học như giới hạn từ và cụm từ cấm vẫn được thực hiện khi lưu chương.

## Cấu trúc output

```text
output/novel/
├── chapters/
├── drafts/
├── reviews/
├── summaries/
└── meta/
    ├── progress.json
    ├── checkpoints.jsonl
    └── sessions/
```

## Theo dõi upstream

Workflow [`Upstream monitor`](.github/workflows/upstream-monitor.yml) kiểm tra repo tác giả [`voocel/ainovel-cli`](https://github.com/voocel/ainovel-cli), nhánh `main`, mỗi ngày. Khi phát hiện commit mới, workflow cập nhật một issue theo dõi duy nhất trong repo này; workflow không tự động merge thay đổi upstream vì bản tiếng Việt có thể cần đồng bộ chọn lọc.

Có thể fetch upstream thủ công để review:

```bash
git remote add upstream https://github.com/voocel/ainovel-cli.git
git fetch upstream main
git log --oneline upstream/main
```

## Build từ source

```bash
git clone https://github.com/JustinNguyen9979/ainovel-cli.git
cd ainovel-cli
go build -trimpath -o ainovel-cli ./cmd/ainovel-cli
./ainovel-cli
```

## Release dành cho maintainer

1. Mở Pull Request vào `main`; không push thẳng code chưa kiểm tra.
2. Đồng bộ version trong `package.json` và `package-lock.json`.
3. Chờ toàn bộ required checks: format, vet, test, race, coverage, benchmark, security và native matrix.
4. Tạo tag SemVer từ commit đã merge, ví dụ:

   ```bash
   git tag v0.8.0
   git push origin v0.8.0
   ```

5. GitHub Actions build một lần sáu native archive, kiểm tra checksum, tạo SBOM/provenance/signature và chờ approval trong environment `production`.
6. Sau approval, publish GitHub Release rồi publish package `ainovel-cli-vn` bằng `NPM_TOKEN` hoặc npm Trusted Publishing OIDC.
7. Chạy post-release smoke test bằng `npm install`, `npm exec` và `--version` trên sáu target.

`NPM_TOKEN` chỉ được lưu trong GitHub repository/environment secret, không đưa vào source hoặc log. npm version đã publish là immutable; rollback phải dùng version mới và có thể deprecate version lỗi.

## Giấy phép

Apache License 2.0 — xem [`LICENSE`](LICENSE).
