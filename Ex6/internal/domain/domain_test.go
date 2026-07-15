package domain

import "testing"

func TestComputeSummary(t *testing.T) {
	tests := []struct {
		name       string
		results    []CheckResult
		durationMS int64
		want       BatchSummary
	}{
		{
			name:       "lot vide",
			results:    nil,
			durationMS: 0,
			want:       BatchSummary{Total: 0, Up: 0, Down: 0, DurationMS: 0},
		},
		{
			name: "toutes accessibles",
			results: []CheckResult{
				{URL: "https://a.com", OK: true},
				{URL: "https://b.com", OK: true},
			},
			durationMS: 120,
			want:       BatchSummary{Total: 2, Up: 2, Down: 0, DurationMS: 120},
		},
		{
			name: "mix up et down",
			results: []CheckResult{
				{URL: "https://go.dev", OK: true},
				{URL: "https://invalid", OK: false, Error: "dns error"},
			},
			durationMS: 812,
			want:       BatchSummary{Total: 2, Up: 1, Down: 1, DurationMS: 812},
		},
		{
			name: "toutes en echec",
			results: []CheckResult{
				{URL: "https://x.invalid", OK: false},
				{URL: "https://y.invalid", OK: false},
				{URL: "https://z.invalid", OK: false},
			},
			durationMS: 500,
			want:       BatchSummary{Total: 3, Up: 0, Down: 3, DurationMS: 500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeSummary(tt.results, tt.durationMS)
			if got != tt.want {
				t.Errorf("ComputeSummary() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
