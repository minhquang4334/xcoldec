# xcoldec 
xcoldec implements decoding of xlsx column per row to golang structure. 

# Installation
```
go get github/minhquang4334/xcoldec
```

# Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/minhquang4334/xcoldec"

	"github.com/xuri/excelize/v2"
)

type EmbeddedStruct struct {
	ColumnK string `col:"K"`
	ColumnL int    `col:"L"`
}

type SubStruct struct {
	ColJ         int    `col:"J"`
	ColK         string `col:"K"`
}

type SupportedType struct {
  Str      string    `col:"A,omitempty"`
  Int      int       `col:"B"` // should not be empty
  Uint     uint      `col:"C"`
  Boolean  bool      `col:"D"`
  Float32  float32   `col:"E"`
  Float64  float64   `col:"F"`
  StrSlice []string  `col:"G"`
  IntSlice []int     `col:"H"`
  Time     time.Time `col:"I"`
  EmbeddedStruct
  SubStruct SubStruct
}


func main() {
	f, err := excelize.OpenFile("sample.xlsx")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, row := range rows[1:] {
		hoge := SupportedType{}
		dec := xcoldec.NewDecoder(row)
		if err := dec.Decode(&hoge); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%+v\n", hoge)
	}
}
```
