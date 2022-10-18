package xcoldec

import (
	"testing"
	"time"

	"github.com/araddon/dateparse"
	"github.com/google/go-cmp/cmp"
)

type testStruct struct {
	Str      string    `column:"A"`
	Int      int       `column:"B"`
	Uint     uint      `column:"C"`
	Boolean  bool      `column:"D"`
	Float32  float32   `column:"E"`
	Float64  float64   `column:"F"`
	StrSlice []string  `column:"G"`
	IntSlice []int     `column:"H"`
	Time     time.Time `column:"I"`
}

func TestUnmarshal(t *testing.T) {
	timeStr := "2021-08-15 02:30:45"
	timeData, _ := dateparse.ParseAny(timeStr)
	testCases := []struct {
		name    string
		row     []string
		want    *testStruct
		wantErr bool
	}{
		{
			"ok with all data type",
			[]string{"str", "64", "128", "true", "3.14", "6.28", "a,b,c", "1,2,3", "2021-08-15 02:30:45"},
			&testStruct{Str: "str", Int: 64, Uint: 128, Boolean: true, Float32: 3.14, Float64: 6.28, StrSlice: []string{"a", "b", "c"}, IntSlice: []int{1, 2, 3}, Time: timeData},
			false,
		},
		{
			"invalid int type",
			[]string{"str", "invalid"},
			&testStruct{Str: "str"},
			true,
		},
		{
			"invalid boolean type",
			[]string{"str", "64", "128", "invalid"},
			&testStruct{Str: "str", Int: 64, Uint: 128},
			true,
		},
		{
			"invalid float type",
			[]string{"str", "64", "128", "true", "invalid"},
			&testStruct{Str: "str", Int: 64, Uint: 128, Boolean: true},
			true,
		},
		{
			"invalid int slice type",
			[]string{"str", "64", "128", "true", "3.14", "6.28", "a,b,c", "invalid, invalid"},
			&testStruct{Str: "str", Int: 64, Uint: 128, Boolean: true, Float32: 3.14, Float64: 6.28, StrSlice: []string{"a", "b", "c"}},
			true,
		},
		{
			"invalid time.Time type",
			[]string{"str", "64", "128", "true", "3.14", "6.28", "a,b,c", "1,2,3", "invalid"},
			&testStruct{Str: "str", Int: 64, Uint: 128, Boolean: true, Float32: 3.14, Float64: 6.28, StrSlice: []string{"a", "b", "c"}, IntSlice: []int{1, 2, 3}},
			true,
		},
	}

	for _, tc := range testCases {
		dec := NewDecoder(tc.row)
		var got testStruct
		err := dec.Decode(&got)
		gotErr := err != nil
		if gotErr != tc.wantErr {
			t.Fatalf("wantErr=%v but got=%v", tc.wantErr, gotErr)
		}
		if diff := cmp.Diff(tc.want, &got); diff != "" {
			t.Errorf("-want, +got:\n%s", diff)
		}
	}
}

type Anonymous struct {
	ColumnF string `column:"F"`
	ColumnG int    `column:"G"`
}

type arbitraryStruct struct {
	Str string `column:"B"`
	Int int    `column:"E"`
	Anonymous
	Boolean bool `column:"H"`
}

func TestUnmarshal_Arbitrary(t *testing.T) {
	row := []string{"", "str", "", "", "32", "anonymous", "123", "true", ""}
	dec := NewDecoder(row)
	var got arbitraryStruct
	err := dec.Decode(&got)
	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	var want = arbitraryStruct{
		Str: "str",
		Int: 32,
		Anonymous: Anonymous{
			ColumnF: "anonymous",
			ColumnG: 123,
		},
		Boolean: true,
	}

	if diff := cmp.Diff(&want, &got); diff != "" {
		t.Errorf("-want, +got:\n%s", diff)
	}
}
