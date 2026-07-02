package workbook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// writeTestWorkbook creates a minimal 2-sheet advertiser workbook fixture.
func writeTestWorkbook(t *testing.T) string {
	t.Helper()
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", "campaigns")
	for i, h := range []string{"campaign_name", "budget_max"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("campaigns", cell, h)
	}
	f.SetCellValue("campaigns", "A2", "01_학습자료")
	f.SetCellValue("campaigns", "B2", 25000)
	f.NewSheet("브리프")
	f.SetCellValue("브리프", "A1", "상품명")
	f.SetCellValue("브리프", "B1", "대표 랜딩 URL")
	f.SetCellValue("브리프", "A2", "캐츠잉글리시")
	f.SetCellValue("브리프", "B2", "https://www.catsenglish.net")
	p := filepath.Join(t.TempDir(), "input.xlsx")
	if err := f.SaveAs(p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInspectDumpsSheets(t *testing.T) {
	d, err := Inspect(writeTestWorkbook(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Sheets) != 2 {
		t.Fatalf("sheets = %d, want 2", len(d.Sheets))
	}
	c := d.Sheets[0]
	if c.Name != "campaigns" || c.Headers[0] != "campaign_name" {
		t.Fatalf("unexpected first sheet: %+v", c)
	}
	if c.Rows[0]["campaign_name"] != "01_학습자료" {
		t.Errorf("row value mismatch: %+v", c.Rows[0])
	}
	if d.Sheets[1].Rows[0]["대표 랜딩 URL"] != "https://www.catsenglish.net" {
		t.Errorf("brief row mismatch: %+v", d.Sheets[1].Rows[0])
	}
}

func TestInspectRejectsDRM(t *testing.T) {
	p := filepath.Join(t.TempDir(), "drm.xlsx")
	if err := os.WriteFile(p, []byte("SCDSA fake drm container"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Inspect(p)
	if err == nil || !strings.Contains(err.Error(), "DRM") {
		t.Fatalf("want DRM error, got %v", err)
	}
}

func TestInspectRejectsEncryptedCFB(t *testing.T) {
	p := filepath.Join(t.TempDir(), "encrypted.xlsx")
	if err := os.WriteFile(p, append([]byte{0xD0, 0xCF, 0x11, 0xE0}, []byte("fake cfb")...), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Inspect(p)
	if err == nil || !strings.Contains(err.Error(), "암호화") {
		t.Fatalf("want encrypted-CFB error, got %v", err)
	}
	if strings.Contains(err.Error(), "DRM") {
		t.Fatalf("CFB error must be distinguishable from DRM error, got %v", err)
	}
}

func TestInspectRejectsNonZip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "junk.xlsx")
	if err := os.WriteFile(p, []byte("hello world not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Inspect(p)
	if err == nil || !strings.Contains(err.Error(), "xlsx(ZIP) 형식이 아닙니다") {
		t.Fatalf("want non-zip format error, got %v", err)
	}
}
