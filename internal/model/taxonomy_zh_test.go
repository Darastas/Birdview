package model

import (
	"testing"
	"time"
)

func TestOrderAndFamilyZh(t *testing.T) {
	if got := OrderZh("Coraciiformes"); got != "佛法僧目" {
		t.Fatalf("got %q, want 佛法僧目", got)
	}
	if got := FamilyZh("Alcedinidae", "Kingfishers"); got != "翠鸟科" {
		t.Fatalf("got %q, want 翠鸟科", got)
	}
	if got := FamilyZh("", "Herons, Egrets, and Bitterns"); got != "鹭科" {
		t.Fatalf("got %q, want 鹭科", got)
	}
	if got := FamilyZh("CustomFam", "戴胜科"); got != "戴胜科" {
		t.Fatalf("should preserve existing CJK family name, got %q", got)
	}
	line := FormatTaxonLine(Taxon{
		Order:         "Passeriformes",
		FamilySciName: "Muscicapidae",
		FamilyComName: "Old World Flycatchers",
	})
	if line != "雀形目 · 鹟科" {
		t.Fatalf("got %q, want 雀形目 · 鹟科", line)
	}
}

func TestQualityAndObsTimeFormatting(t *testing.T) {
	if got := QualityGradeZh("research"); got != "研究级" {
		t.Fatalf("got %q", got)
	}
	if got := QualityGradeZh("needs_id"); got != "待复核" {
		t.Fatalf("got %q", got)
	}

	now := time.Date(2026, 11, 25, 19, 0, 0, 0, time.Local)
	if got := formatObsTimeAt("2026-11-25 08:30", now); got != "今天 08:30" {
		t.Fatalf("got %q, want 今天 08:30", got)
	}
	if got := formatObsTimeAt("2026-11-24 16:45", now); got != "昨天 16:45" {
		t.Fatalf("got %q, want 昨天 16:45", got)
	}
	if got := formatObsTimeAt("2026-11-10", now); got != "11月10日" {
		t.Fatalf("got %q, want 11月10日", got)
	}
	if got := formatObsTimeAt("2025-10-02 07:15", now); got != "2025年10月2日 07:15" {
		t.Fatalf("got %q", got)
	}
}
