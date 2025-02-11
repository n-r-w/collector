//go:build xotpl

package gotpl

import (
	"strings"

	"github.com/kenshaw/snaker"
	xo "github.com/xo/xo/types"
)

/* English:
If there are missing types, they can be added:
- For arrays in map pgPGXArrMapping
- For regular types in pgxPostgresGoTypeHelper
*/

var pgPGXArrMapping = map[string]string{
	"bool":          "[]bool",
	"[]byte":        "[][]byte",
	"float64":       "[]float64",
	"float32":       "[]float32",
	"int":           "[]int",
	"int8":          "[]int8",
	"int16":         "[]int16",
	"int64":         "[]int64",
	"int32":         "[]int32",
	"uint":          "[]uint",
	"uint8":         "[]uint8",
	"uint16":        "[]uint16",
	"uint64":        "[]uint64",
	"uint32":        "[]uint32",
	"string":        "[]string",
	"uuid.UUID":     "[]uuid.UUID",
	"time.Time":     "[]time.Time",
	"time.Duration": "[]time.Duration",
	"net.IPNet":     "[]net.IPNet",
}

func pgxPostgresGoType(d xo.Type, schema, itype, _ string) (string, string, error) {
	goType, zero, err := pgxPostgresGoTypeHelper(d, schema, itype)
	if err != nil {
		return "", "", err
	}
	if d.IsArray {
		arrType, ok := pgPGXArrMapping[goType]
		goType, zero = "[]byte", "nil"
		if ok {
			goType = arrType
		}
	}
	return goType, zero, nil
}

func pgxPostgresGoTypeHelper(d xo.Type, schema, itype string) (string, string, error) {
	// SETOF -> []T
	if strings.HasPrefix(d.Type, "SETOF ") {
		d.Type = d.Type[len("SETOF "):]
		goType, _, err := pgxPostgresGoTypeHelper(d, schema, itype)
		if err != nil {
			return "", "", err
		}
		return "[]" + goType, "nil", nil
	}
	// If it's an array, the underlying type shouldn't also be set as an array
	typNullable := d.Nullable && !d.IsArray
	// special type handling
	typ := d.Type
	switch {
	case typ == `"char"`:
		typ = "char"
	case strings.HasPrefix(typ, "information_schema."):
		switch strings.TrimPrefix(typ, "information_schema.") {
		case "cardinal_number":
			typ = "integer"
		case "character_data", "sql_identifier", "yes_or_no":
			typ = "character varying"
		case "time_stamp":
			typ = "timestamp with time zone"
		}
	}
	var goType, zero string
	switch typ {
	case "boolean":
		goType, zero = "bool", "false"
		if typNullable {
			goType, zero = "pgtype.Bool", "pgtype.Bool{}"
		}
	case "bpchar", "character varying", "character", "money", "text", "name":
		goType, zero = "string", `""`
		if typNullable {
			goType, zero = "pgtype.Text", "pgtype.Text{}"
		}
	case "inet":
		goType, zero = "net.IPNet", `""`
		if typNullable {
			goType, zero = "pgtype.Inet", "pgtype.Inet{}"
		}
	case "smallint":
		goType, zero = "int16", "0"
		if typNullable {
			goType, zero = "pgtype.Int2", "pgtype.Int2{}"
		}
	case "integer":
		goType, zero = itype, "0"
		if typNullable {
			goType, zero = "pgtype.Int4", "pgtype.Int4{}"
		}
	case "bigint":
		goType, zero = "int64", "0"
		if typNullable {
			goType, zero = "pgtype.Int8", "pgtype.Int8{}"
		}
	case "oid":
		goType, zero = "uint32", "0"
		if typNullable {
			goType, zero = "pgtype.Uint32", "pgtype.Uint32{}"
		}
	case "real":
		goType, zero = "float32", "0.0"
		if typNullable {
			goType, zero = "pgtype.Float4", "pgtype.Float4{}"
		}
	case "double precision", "numeric":
		goType, zero = "float64", "0.0"
		if typNullable {
			goType, zero = "pgtype.Float8", "pgtype.Float8{}"
		}
	case "date":
		goType, zero = "time.Time", "time.Time{}"
		if typNullable {
			goType, zero = "pgtype.Date", "pgtype.Date{}"
		}
	case "timestamp with time zone":
		goType, zero = "time.Time", "time.Time{}"
		if typNullable {
			goType, zero = "pgtype.Timestamptz", "pgtype.Timestamptz{}"
		}
	case "timestamp without time zone":
		goType, zero = "time.Time", "time.Time{}"
		if typNullable {
			goType, zero = "pgtype.Timestamp", "pgtype.Timestamp{}"
		}
	case "time with time zone":
		goType, zero = "time.Time", "time.Time{}"
		if typNullable {
			goType, zero = "pgtype.Timestamptz", "pgtype.Timestamptz{}"
		}
	case "time without time zone":
		goType, zero = "time.Time", "time.Time{}"
		if typNullable {
			goType, zero = "pgtype.Time", "pgtype.Time{}"
		}
	case "bit":
		goType, zero = "uint8", "0"
		if typNullable {
			goType, zero = "*uint8", "nil"
		}
	case "any", "xml":
		goType, zero = "[]byte", "nil"
	case "bit varying":
		goType, zero = "[]byte", "nil"
		if typNullable {
			goType, zero = "pgtype.Bits", "pgtype.Bits{}"
		}
	case "bytea":
		goType, zero = "[]byte", "nil"
		if typNullable {
			goType, zero = "[]byte", "nil"
		}
	case "interval":
		goType, zero = "time.Duration", "0"
		if typNullable {
			goType, zero = "pgtype.Interval", "pgtype.Interval{}"
		}
	case "json":
		goType, zero = "[]byte", "nil"
		if typNullable {
			goType, zero = "[]byte", "nil"
		}
	case "jsonb":
		goType, zero = "[]byte", "nil"
		if typNullable {
			goType, zero = "[]byte", "nil"
		}
	case "hstore":
		goType, zero = "pgtype.Hstore", "pgtype.Hstore{}"
	case "uuid":
		goType, zero = "uuid.UUID", "uuid.UUID{}"
		if typNullable {
			goType, zero = "pgtype.UUID", "pgtype.UUID{}"
		}
	default:
		goType, zero = schemaType(d.Type, typNullable, schema)
	}
	return goType, zero, nil
}

func schemaType(typ string, nullable bool, schema string) (string, string) {
	if strings.HasPrefix(typ, schema+".") {
		// in the same schema, so chop off
		typ = typ[len(schema)+1:]
	}
	if nullable {
		typ = "null_" + typ
	}
	s := snaker.SnakeToCamelIdentifier(typ)
	return s, s + "{}"
}
