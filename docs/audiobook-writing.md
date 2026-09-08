# Hướng dẫn Sáng tác Tiểu thuyết Tối ưu cho Audiobook và Truyện Nghe

Tài liệu này hướng dẫn cách sử dụng và các nguyên tắc đằng sau phong cách sáng tác **Audiobook và Truyện Nghe** (`audiobook`) vừa được tích hợp vào `ainovel-cli`.

Khi chuyển đổi từ **đọc** (Visual) sang **nghe** (Auditory), trải nghiệm tiếp nhận thông tin của thính giả thay đổi hoàn toàn. Hệ thống cần điều chỉnh văn phong, cấu trúc câu và nhịp độ để câu chuyện sinh động, dễ tiếp thu và lôi cuốn hơn khi được đọc thành tiếng (bởi MC hoặc công cụ Text-to-Speech - TTS).

---

## 1. Cách Kích Hoạt Phong Cách Audiobook

Để áp dụng phong cách này cho tác phẩm của bạn, hãy cập nhật cấu hình trong file `config.json` (hoặc cấu hình thông qua TUI):

```json
{
  "style": "audiobook"
}
```

Hệ thống sẽ tự động tải các tệp cấu hình phong cách và mẫu cốt truyện tích hợp sẵn:
- **Style Preset**: [assets/styles/audiobook.md](../assets/styles/audiobook.md)
- **Style References**: [assets/references/genres/audiobook/style-references.md](../assets/references/genres/audiobook/style-references.md)
- **Arc Templates**: [assets/references/genres/audiobook/arc-templates.md](../assets/references/genres/audiobook/arc-templates.md)

---

## 2. Các Nguyên Tắc Cốt Lõi Khi Viết Cho Người Nghe

### A. Rõ Ràng Về Thính Giác (Auditory Clarity)
- **Hạn chế từ Hán-Việt cổ/hiếm**: Những từ như *phụ phụ* (vợ chồng), *tỷ đệ* (chị em), *quy quyền* (về nước),... có thể gây khó hiểu cho người nghe khi lướt qua bằng tai. Thay bằng từ thuần Việt hoặc từ Hán-Việt quen thuộc.
- **Tránh từ đồng âm gây bối rối**: Tránh các cặp từ phát âm giống nhau nhưng nghĩa khác xa nhau trong cùng một ngữ cảnh hẹp để tránh gây hiểu lầm.
- **Nhịp thở của câu**: Câu văn viết có thể dài, nhưng câu đọc thành tiếng phải có điểm ngắt nghỉ. Một câu quá dài sẽ khiến người nghe mất tập trung hoặc mệt mỏi.
  - *Chưa tốt*: *"Trong ánh chiều tà đang dần tắt lịm phía sau dãy núi xa xăm bao phủ bởi sương mù dày đặc, Lâm đứng nhìn ngôi nhà tranh cũ kỹ mà anh đã từng gắn bó suốt cả tuổi thơ đầy gian khó của mình."*
  - *Tốt hơn*: *"Ánh chiều tà dần tắt sau dãy núi mờ sương. Lâm đứng lặng, nhìn ngôi nhà tranh cũ kỹ. Nơi đây đã gắn bó với anh suốt thời thơ ấu đầy gian khó."*

### B. Chuyển Cảnh Tường Minh Bằng Ngôn Từ (Auditory Transitions)
- Người đọc sách giấy có thể nhận biết chuyển cảnh bằng dòng trống hoặc dấu gạch ngang. Người nghe thì không.
- **Quy tắc**: Mỗi khi chuyển cảnh (thay đổi thời gian hoặc địa điểm), chương truyện **bắt buộc** phải sử dụng từ ngữ chuyển tiếp rõ ràng ngay đầu đoạn văn mới để dẫn dắt thính giác của người nghe.
  - *Chưa tốt*: *(Đoạn trước ở phòng làm việc, đoạn sau đột ngột nhảy sang quán trà mà không có câu dẫn)*
  - *Tốt hơn*: *"Hai tiếng sau, tại quán trà nhỏ ở góc phố, Lâm ngồi đối diện với một người đàn ông lạ mặt..."*

### C. Lời Thoại Khẩu Ngữ Việt Tự Nhiên (Spoken Dialogue)
- Nhân vật nói chuyện phải mang tính chất hội thoại thực tế của người Việt, có sự ngắt quãng, từ cảm thán và trợ từ tình thái cuối câu (`hả`, `nhé`, `à`, `đâu`, `chứ`, `sao`, `đấy`, `nha`, `nhỉ`).
- Cắt bỏ các câu thoại quá đầy đủ ngữ pháp kiểu văn viết học thuật hoặc dịch thô.
  - *Chưa tốt*: *"Tôi thực sự khuyên bạn không nên bước vào căn phòng đó vào thời điểm nguy hiểm này."*
  - *Tốt hơn*: *"Đừng vào đấy, nguy hiểm lắm!"*

### D. Tăng Cường Kích Thích Thính Giác (Show, Don't Tell via Sound)
- **Sử dụng từ tượng thanh**: Thay vì viết *"Có âm thanh lạ vang lên"*, hãy mô tả trực tiếp âm thanh đó bằng từ tượng thanh sinh động để kích thích trí tưởng tượng của thính giả: *"Một tiếng 'cạch' nhẹ vang lên"*, *"Tiếng xích sắt kéo lê trên sàn nhà kêu lẹt xẹt"*.
- **Hạn chế độc thoại nội tâm quá dài**: Độc thoại nội tâm không đi kèm hành động thực tế sẽ tạo cảm giác đơn điệu, dễ làm người nghe buồn ngủ hoặc xao nhãng. Hãy biến suy nghĩ thành hành động cụ thể hoặc lời nói.

---

## 3. Cách Tinh Chỉnh Quy Tắc Cơ Học Cho Trình Đọc/TTS Cụ Thể

Nếu bạn sử dụng công cụ TTS (Text-to-Speech) để chuyển truyện thành giọng đọc, một số từ ngữ có thể bị phát âm sai hoặc bị ngọng tùy thuộc vào bộ đọc (giọng miền Bắc, miền Nam hoặc giọng AI của các nền tảng khác nhau).

Bạn có thể cấu hình danh sách cấm hoặc cảnh báo riêng bằng cách tạo file `.md` trong thư mục `.ainovel/rules/` của dự án (ví dụ: `.ainovel/rules/audiobook-rules.md`):

```markdown
---
# Cấu hình kiểm tra cơ học tối ưu cho công cụ đọc thành tiếng
forbidden_phrases:
  - "nghĩ thầm trong đầu"   # Lặp từ thừa thãi khi nghe
  - "cảm xúc lẫn lộn"       # Câu sáo rỗng AI
  - "đáng chú ý là"

# Giới hạn tần suất xuất hiện của một số từ gây mệt mỏi khi nghe nhiều lần
fatigue_words:
  bỗng nhiên: 1
  dường như: 2
  không khỏi: 1
  vài nhịp thở: 2
---

# Quy tắc bổ sung cho Truyện Nghe

- Tránh các mô tả tĩnh về phong cảnh vượt quá 3 câu liên tiếp.
- Đối thoại phải phân biệt rõ ràng tông giọng giữa nhân vật nam (trầm ổn, dứt khoát) và nhân vật nữ (nhẹ nhàng, lanh lợi).
- Mỗi chương phải kết thúc bằng một sự kiện bỏ lửng (cliffhanger) để tạo sự tò mò.
```

---

## 4. Gợi Ý Về Thiết Kế Cung Truyện (Arc Planning)
Đối với Audiobook, cấu trúc **Cung Khởi Đầu Kịch Tính (5-7 chương)** được khuyến nghị sử dụng để nhanh chóng kéo lượng người nghe ở những phút đầu tiên, sau đó duy trì bằng **Cung Xử Lý Xung Đột Nhịp Độ Cao**. Bạn có thể tham khảo chi tiết mẫu tại [assets/references/genres/audiobook/arc-templates.md](../assets/references/genres/audiobook/arc-templates.md).
