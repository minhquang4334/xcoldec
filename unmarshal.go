package xcoldec

import (
	"encoding"
	"errors"
	"fmt"
	"strconv"
	"time"

	"reflect"

	"strings"

	"github.com/araddon/dateparse"
	"github.com/xuri/excelize/v2"
)

const (
	defaultTag = "col"
	sliceDelim = ","
	tagDelim   = ","
)

var (
	nonScalarType             = []string{"time.Time"}
	ErrInvalidPointerOfStruct = errors.New("v must be a pointer of struct")
)

// NewDecoder return a new decoder that read from row.
func NewDecoder(row []string) *Decoder {
	dec := &Decoder{row}

	return dec
}

// Decoder decodes values from row.
type Decoder struct {
	row []string
}

// Decode decodes given row (string slice) to struct of column.
//
// Supported Go data types are:
// - string
// - int, float family
// - Boolean
// - the type implements encoding.TextUnmarshaler
// - slices
//   - the element type must be Decode() support type
//   - element will split by "," from given string
//
// - time.Time
// - embedded struct
// - sub struct
func (d *Decoder) Decode(v interface{}) error {
	vt := reflect.ValueOf(v)
	if vt.IsNil() || !(vt.Kind() == reflect.Pointer && vt.Elem().Kind() == reflect.Struct) {
		return ErrInvalidPointerOfStruct
	}

	if err := decode(d.row, reflect.ValueOf(v).Elem()); err != nil {
		return err
	}

	return nil
}

func decode(row []string, v reflect.Value) error {
	NumField := v.NumField()

	for i := 0; i < NumField; i++ {
		sField := v.Type().Field(i)
		sValue := v.Field(i)
		// recurrent decoding with embedding of struct
		if sField.Anonymous || sField.Type.Kind() == reflect.Struct && !contains(nonScalarType, sField.Type.String()) {
			err := decode(row, sValue)
			if err != nil {
				return err
			}
			continue
		}

		col := parseTag(sField.Tag)
		if col == nil {
			continue
		}

		cellVal, err := getCol(row, col.name)
		if err != nil {
			return err
		}

		if !col.omitEmpty && cellVal == "" {
			return &DecodeError{
				column:       col.name,
				expectedType: sValue.Kind().String(),
				gotValue:     cellVal,
				errMsg: "should not be empty",
			}
		}

		if col.omitEmpty && cellVal == "" {
			continue
		}

		switch kind := sValue.Kind(); kind {
		case reflect.Slice:
			els := strings.Split(cellVal, ",")

			size := len(els)
			// grow slice size
			if size >= sValue.Cap() {
				nv := reflect.MakeSlice(sValue.Type(), sValue.Len(), size)
				reflect.Copy(nv, sValue)
				sValue.Set(nv)
				sValue.SetLen(size)
			}

			for i, el := range els {
				if err := decodeScalar(el, sValue.Index(i), col.name); err != nil {
					sValue.Set(reflect.Zero(sValue.Type()))
					return err
				}
			}

		default:
			if err := decodeScalar(cellVal, sValue, col.name); err != nil {
				return err
			}
		}
	}

	return nil
}

var (
	textUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

// refs: https://github.com/aereal/paramsenc/blob/main/unmarshal.go#L118
func decodeScalar(val string, fieldValue reflect.Value, colName string) error {
	if fieldValue.Type().Implements(textUnmarshaler) {
		fv := fieldValue
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		if v, ok := fv.Interface().(encoding.TextUnmarshaler); ok {
			return v.UnmarshalText([]byte(val))
		}
	}

	expectedType := fieldValue.Kind().String()
	switch kind := fieldValue.Interface(); kind.(type) {
	case string:
		fieldValue.SetString(val)
	case int, int8, int16, int32, int64:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return &DecodeError{
				column:       colName,
				expectedType: expectedType,
				gotValue:     val,
				errMsg:       err.Error(),
			}
		}
		if fieldValue.OverflowInt(n) {
			return &DecodeError{
				column:       colName,
				expectedType: expectedType,
				gotValue:     val,
				errMsg:       "overflow int size",
			}
		}
		fieldValue.SetInt(n)
	case uint, uint8, uint16, uint32, uint64:
		n, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return &DecodeError{
				column:       colName,
				expectedType: expectedType,
				gotValue:     val,
				errMsg:       err.Error(),
			}
		}
		if fieldValue.OverflowUint(n) {
			return &DecodeError{
				column:       colName,
				expectedType: expectedType,
				gotValue:     val,
				errMsg:       "overflow uint size",
			}
		}
		fieldValue.SetUint(n)
	case float32, float64:
		n, err := strconv.ParseFloat(val, fieldValue.Type().Bits())
		if err != nil {
			return &DecodeError{
				column:       colName,
				expectedType: expectedType,
				gotValue:     val,
				errMsg:       err.Error(),
			}
		}
		if fieldValue.OverflowFloat(n) {
			return &DecodeError{
				column:       colName,
				expectedType: expectedType,
				gotValue:     val,
				errMsg:       "overflow float size",
			}
		}
		fieldValue.SetFloat(n)
	case bool:
		switch val {
		case "true", "1":
			fieldValue.SetBool(true)
		case "false", "0", "":
			fieldValue.SetBool(false)
		default:
			return &DecodeError{
				column:       colName,
				expectedType: expectedType,
				gotValue:     val,
				errMsg:       "invalid boolean value, value should be in (0, 1, true, false)",
			}
		}
	case time.Time:
		n, err := dateparse.ParseAny(val)
		if err != nil {
			return &DecodeError{
				column:       colName,
				expectedType: expectedType,
				gotValue:     val,
				errMsg:       err.Error(),
			}
		}
		fieldValue.Set(reflect.ValueOf(n))

	default:
		return &DecodeError{
			column:       colName,
			expectedType: expectedType,
			gotValue:     val,
			errMsg:       "unknown error, cannot unmarshal",
		}
	}
	return nil
}

type option struct {
	name      string
	omitEmpty bool
}

func parseTag(tag reflect.StructTag) *option {
	val, ok := tag.Lookup(defaultTag)
	if !ok || val == "" {
		return nil
	}

	res := strings.Split(val, tagDelim)
	colName := res[0]
	omitEmpty := false
	if len(res) > 1 && res[1] == "omitempty" {
		omitEmpty = true
	}

	return &option{name: colName, omitEmpty: omitEmpty}
}

func getCol(row []string, col string) (string, error) {
	c, err := excelize.ColumnNameToNumber(col)
	if err != nil {
		return "", err
	}

	if len(row) < c {
		return "", nil
	}
	return row[c-1], nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

type DecodeError struct {
	column       string
	expectedType string
	gotValue     string
	errMsg       string
}

var _ error = &DecodeError{}

func (e *DecodeError) Error() string {
	return fmt.Sprintf("can not decode value of column %q: expectedType: %q, gotValue: %q, errMsg: %q", e.column, e.expectedType, e.gotValue, e.errMsg)
}
