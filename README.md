# xcoldec 
xcoldec implements decoding of xlsx column to golang structure. 

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
	ColumnK string `column:"K"`
	ColumnL int    `column:"L"`
}

type SupportedType struct {
	Str      string    `column:"A"`
	Int      int       `column:"B"`
	Uint     uint      `column:"C"`
	Boolean  bool      `column:"D"`
	Float32  float32   `column:"E"`
	Float64  float64   `column:"F"`
	StrSlice []string  `column:"G"`
  	IntSlice []int     `column:"H"`
  	Time     time.Time `column:"I"`
  	EmbeddedStruct
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
