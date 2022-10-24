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
)

var nonScalarType = []string{"time.Time"}

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
		return errors.New("v must be a pointer of struct")
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
		fmt.Println(sField.Type, sField.Anonymous, sField.Type.Kind())
		// recurrent decoding with embedding of struct
		if sField.Anonymous || sField.Type.Kind() == reflect.Struct && !contains(nonScalarType, sField.Type.String()) {
			decode(row, sValue)
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
				if err := decodeScalar(el, sValue.Index(i)); err != nil {
					sValue.Set(reflect.Zero(sValue.Type()))
					return err
				}
			}

		default:
			if err := decodeScalar(cellVal, sValue); err != nil {
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
func decodeScalar(val string, fieldValue reflect.Value) error {
	if fieldValue.Type().Implements(textUnmarshaler) {
		fv := fieldValue
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		if v, ok := fv.Interface().(encoding.TextUnmarshaler); ok {
			return v.UnmarshalText([]byte(val))
		}
	}

	switch kind := fieldValue.Interface(); kind.(type) {
	case string:
		fieldValue.SetString(val)
	case int, int8, int16, int32, int64:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot convert value %q to %T: %w", val, kind, err)
		}
		if fieldValue.OverflowInt(n) {
			return fmt.Errorf("cannot convert value %q: overflow int size", val)
		}
		fieldValue.SetInt(n)
	case uint, uint8, uint16, uint32, uint64:
		n, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot convert value %q to %T: %w", val, kind, err)
		}
		if fieldValue.OverflowUint(n) {
			return fmt.Errorf("cannot convert value %q: overflow uint size", val)
		}
		fieldValue.SetUint(n)
	case float32, float64:
		n, err := strconv.ParseFloat(val, fieldValue.Type().Bits())
		if err != nil {
			return fmt.Errorf("cannot convert value %q to %T: %w", val, kind, err)
		}
		if fieldValue.OverflowFloat(n) {
			return fmt.Errorf("cannot convert value %q: overflow float size", val)
		}
		fieldValue.SetFloat(n)
	case bool:
		switch val {
		case "true":
			fieldValue.SetBool(true)
		case "false":
			fieldValue.SetBool(false)
		default:
			return fmt.Errorf("invalid boolean: %s", val)
		}
	case time.Time:
		n, err := dateparse.ParseAny(val)
		if err != nil {
			return fmt.Errorf("cannot convert value %q to %T: %w", val, kind, err)
		}
		fieldValue.Set(reflect.ValueOf(n))

	default:
		return fmt.Errorf("%T is cannot unmarshal", kind)
	}
	return nil
}

type option struct {
	name string
}

func parseTag(tag reflect.StructTag) *option {
	val, ok := tag.Lookup(defaultTag)
	if !ok {
		return nil
	}

	return &option{name: val}
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
