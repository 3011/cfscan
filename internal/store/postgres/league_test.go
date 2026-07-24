package postgres

import "testing"

func TestNormalizeLeaguePage(t *testing.T) {
	tests := []struct {
		name                  string
		total, page, pageSize int
		wantPage, wantPages   int
		wantOffset            int
	}{
		{name: "first page", total: 2926, page: 1, pageSize: 50, wantPage: 1, wantPages: 59, wantOffset: 0},
		{name: "second page", total: 2926, page: 2, pageSize: 50, wantPage: 2, wantPages: 59, wantOffset: 50},
		{name: "legacy five hundred", total: 2926, page: 2, pageSize: 500, wantPage: 2, wantPages: 6, wantOffset: 500},
		{name: "clamps beyond last", total: 51, page: 9, pageSize: 50, wantPage: 2, wantPages: 2, wantOffset: 50},
		{name: "empty", total: 0, page: 9, pageSize: 50, wantPage: 1, wantPages: 0, wantOffset: 0},
		{name: "normalizes invalid", total: 80, page: 0, pageSize: 501, wantPage: 1, wantPages: 2, wantOffset: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, pages, offset := normalizeLeaguePage(test.total, test.page, test.pageSize)
			if page != test.wantPage || pages != test.wantPages || offset != test.wantOffset {
				t.Fatalf("got page=%d pages=%d offset=%d", page, pages, offset)
			}
		})
	}
}
