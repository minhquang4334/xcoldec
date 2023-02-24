package xcoldec_test

import (
	"testing"
	"time"

	"github.com/araddon/dateparse"
	"github.com/google/go-cmp/cmp"
	"github.com/minhquang4334/xcoldec"
)

type testStruct struct {
	Str      string    `col:"A"`
	Int      int       `col:"B"`
	Uint     uint      `col:"C"`
	Boolean  bool      `col:"D"`
	Float32  float32   `col:"E"`
	Float64  float64   `col:"F"`
	StrSlice []string  `col:"G"`
	IntSlice []int     `col:"H"`
	Time     time.Time `col:"I"`
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
			[]string{"str", "64", "128", "true", "3.14", "6.28", "a,b,c", "1,2,3", timeStr},
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
		t.Run(tc.name, func(t *testing.T) {
			dec := xcoldec.NewDecoder(tc.row)
			var got testStruct
			err := dec.Decode(&got)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("wantErr=%v but got=%v", tc.wantErr, gotErr)
			}
			if diff := cmp.Diff(tc.want, &got); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}

type arbitraryStruct struct {
	Str     string `col:"B"`
	Int     int    `col:"E"`
	Boolean bool   `col:"H"`
}

func TestUnmarshal_Arbitrary(t *testing.T) {
	row := []string{"", "str", "", "", "32", "", "", "true", ""}
	dec := xcoldec.NewDecoder(row)
	var got arbitraryStruct
	err := dec.Decode(&got)
	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	var want = arbitraryStruct{
		Str:     "str",
		Int:     32,
		Boolean: true,
	}

	if diff := cmp.Diff(&want, &got); diff != "" {
		t.Errorf("-want, +got:\n%s", diff)
	}
}

type EmbeddedStruct struct {
	ColumnC string `col:"C"`
	ColumnD int    `col:"D"`
}

type SubSubStruct struct {
	ColE int    `col:"E"`
	ColF string `col:"F"`
}

type SubStruct struct {
	ColA         int    `col:"A"`
	ColB         string `col:"B"`
	SubSubStruct SubSubStruct
}

type Embedded struct {
	Sub SubStruct
	EmbeddedStruct
}

func TestUnmarshal_Embedded(t *testing.T) {
	row := []string{"32", "str", "str2", "64", "128", "str3"}
	dec := xcoldec.NewDecoder(row)
	var got Embedded
	err := dec.Decode(&got)
	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	var want = Embedded{
		Sub: SubStruct{
			ColA: 32,
			ColB: "str",
			SubSubStruct: SubSubStruct{
				ColE: 128,
				ColF: "str3",
			},
		},
		EmbeddedStruct: EmbeddedStruct{
			ColumnC: "str2",
			ColumnD: 64,
		},
	}

	if diff := cmp.Diff(&want, &got); diff != "" {
		t.Errorf("-want, +got:\n%s", diff)
	}
}

type omitEmptyStruct struct {
	Int     int    `col:"A,omitempty"`
	Str     string `col:"B"`
	Boolean bool   `col:"C"`
}

func TestUnmarshal_OmitEmpty(t *testing.T) {
	testCases := []struct {
		name    string
		row     []string
		want    *omitEmptyStruct
		wantErr bool
	}{
		{
			"omit empty ok",
			[]string{"", "str", "true"},
			&omitEmptyStruct{
				Str:     "str",
				Boolean: true,
			},
			false,
		},
		{
			"not omit empty",
			[]string{"2", "", "true"},
			&omitEmptyStruct{Int: 2},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dec := xcoldec.NewDecoder(tc.row)
			var got omitEmptyStruct
			err := dec.Decode(&got)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("wantErr=%v but got=%v, err=%v", tc.wantErr, gotErr, err)
			}
			if diff := cmp.Diff(tc.want, &got); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}

type testBooleanStruct struct {
	False1 bool `col:"A,omitempty"`
	False2 bool `col:"B"`
	False3 bool `col:"C"`
	True1  bool `col:"D"`
	True2  bool `col:"E"`
}

func TestUnmarshal_Boolean(t *testing.T) {
	testStr := []string{"", "false", "0", "true", "1"}

	dec := xcoldec.NewDecoder(testStr)
	var got testBooleanStruct
	err := dec.Decode(&got)
	if err != nil {
		t.Fatalf("error occurred: %v", err)
	}

	var want = testBooleanStruct{
		False1: false,
		False2: false,
		False3: false,
		True1:  true,
		True2:  true,
	}

	if diff := cmp.Diff(&want, &got); diff != "" {
		t.Errorf("-want, +got:\n%s", diff)
	}
}
