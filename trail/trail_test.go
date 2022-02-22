package trail

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDatePaths(t *testing.T) {
	tests := []struct {
		opt       Option
		after1Day bool
		want      []string
		wantErr   bool
	}{
		{Option{DatePath: "2022/02/22"}, false, []string{"2022/02/22"}, false},
		{Option{DatePath: "2022/02/22"}, true, []string{"2022/02/22", "2022/02/23"}, false},
		{Option{DatePath: "2022/02/28"}, true, []string{"2022/02/28", "2022/03/01"}, false},
		{Option{StartDatePath: "2022/02/22", EndDatePath: "2022/02/24"}, false, []string{"2022/02/22", "2022/02/23", "2022/02/24"}, false},
		{Option{StartDatePath: "2022/02/22", EndDatePath: "2022/02/24"}, true, []string{"2022/02/22", "2022/02/23", "2022/02/24", "2022/02/25"}, false},
		{Option{StartDatePath: "2022/02/27", EndDatePath: "2022/03/02"}, false, []string{"2022/02/27", "2022/02/28", "2022/03/01", "2022/03/02"}, false},
	}
	for _, tt := range tests {
		got, err := datePaths(tt.opt, tt.after1Day)
		if err != nil {
			if tt.wantErr {
				continue
			}
			t.Error(err)
			continue
		}
		if tt.wantErr {
			t.Error("want error")
			continue
		}
		if diff := cmp.Diff(got, tt.want, nil); diff != "" {
			t.Errorf("%s", diff)
		}
	}
}
