package main

import (
	"strings"
	"testing"
)

func TestFullSourceCSVPreservesQuotesAndMissing(t *testing.T) {
	rows, err := parseCSV(strings.NewReader("id,value,blank\nx,\"a,b;c\",\n"))
	if err != nil || len(rows) != 1 || rows[0]["value"] != "a,b;c" || rows[0]["blank"] != "" {
		t.Fatal("CSV values changed", err)
	}
	for _, data := range []string{"id,id\nx,y\n", "id,value\nx\n"} {
		if _, err := parseCSV(strings.NewReader(data)); err == nil {
			t.Fatal("accepted invalid header/row")
		}
	}
}
func TestFullSourceJSONLRejectsMalformed(t *testing.T) {
	rows, err := parseJSONL(strings.NewReader("{\"interaction_id\":\"I1\",\"isi\":\"x;y\"}\n"))
	if err != nil || len(rows) != 1 || rows[0]["isi"] != "x;y" {
		t.Fatal(err)
	}
	if _, err := parseJSONL(strings.NewReader("{bad}\n")); err == nil {
		t.Fatal("accepted malformed JSON")
	}
}
func TestFullSourceDuplicateKeysRejected(t *testing.T) {
	if err := uniqueRows([]map[string]string{{"id": "x"}, {"id": "x"}}, []string{"id"}); err == nil {
		t.Fatal("accepted duplicate source key")
	}
}

func TestFullSourcePackageLimitAllowsNativeUnlimited(t *testing.T) {
	for _, v := range []string{"tanpa batas", "25", "10"} {
		if err := validatePackageLimit(v); err != nil {
			t.Fatal(err)
		}
	}
	if err := validatePackageLimit("made up"); err == nil {
		t.Fatal("accepted unknown limit")
	}
}
