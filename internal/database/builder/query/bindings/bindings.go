package bindings

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/musiermoore/oksana-vpn-api/internal/database/builder/query/expression"
)

func FormatArgs(str string, args ...any) string {
	return fmt.Sprintf(str, args...)
}

func Bindings(str string, args ...any) string {
	if len(args) == 0 {
		return str
	}

	return fmt.Sprintf(strings.ReplaceAll(str, "?", "%s"), prepareBindings(args...)...)
}

func prepareBindings(args ...any) []any {
	var bindings []any

	for _, item := range args {
		bindings = append(bindings, prepareBinding(item))
	}

	return bindings
}

func prepareBinding(arg any) string {
	switch v := arg.(type) {

	case expression.RawExpression:
		return Bindings(v.Column, v.Args...)

	case []byte:
		return "X'" + hex.EncodeToString(v) + "'"

	case [32]byte:
		return "X'" + hex.EncodeToString(v[:]) + "'"

	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)

	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)

	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)

	case string:
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"

	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"

	case nil:
		return "NULL"

	default:
		return fmt.Sprintf("'%v'", v)
	}
}
