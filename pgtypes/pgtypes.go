package pgtypes

var PgTypeMap = map[string]string{
	// Arrays
	"text[]":                        "pgtypes.StringArray",
	"varchar[]":                     "pgtypes.StringArray",
	"integer[]":                     "pgtypes.Int32Array",
	"int4[]":                        "pgtypes.Int32Array",
	"int8[]":                        "pgtypes.Int64Array",
	"bigint[]":                      "pgtypes.Int64Array",
	"bool[]":                        "pgtypes.BoolArray",
	"boolean[]":                     "pgtypes.BoolArray",
	"uuid[]":                        "pgtypes.UUIDArray",
	"float8[]":                      "pgtypes.Float64Array",
	"double precision[]":            "pgtypes.Float64Array",
	"timestamptz[]":                 "pgtypes.TimeArray",
	"timestamp[]":                   "pgtypes.TimeArray",
	"timestamp with time zone[]":    "pgtypes.TimeArray",
	"timestamp without time zone[]": "pgtypes.TimeArray",

	// Intervals
	"interval":   "pgtypes.Duration",
	"interval[]": "pgtypes.DurationArray",

	// Boolean
	"bool": "bool",

	// Integers
	"int2": "int16",
	"int4": "int32",
	"int8": "int64",

	// Floating point
	"float4": "float32",
	"float8": "float64",

	// Exact numeric (arbitrary precision)
	// Safer as string unless you adopt a decimal library
	"numeric": "string",

	// Character / text
	"text":    "string",
	"varchar": "string",
	"bpchar":  "string", // CHAR(n)
	"char":    "string", // internal single-byte char
	"name":    "string",

	// Binary
	"bytea": "[]byte",

	// UUID
	// If you don't want external deps, change to "string"
	"uuid": "uuid.UUID",

	// JSON
	"json":  "json.RawMessage",
	"jsonb": "json.RawMessage",

	// XML
	"xml": "string",

	// Date & time
	"date":        "time.Time",
	"timestamp":   "time.Time",
	"timestamptz": "time.Time",

	// Time-of-day types (avoid fake dates)
	"time":   "string",
	"timetz": "string",

	// Network
	"inet":     "net.IPNet",
	"cidr":     "net.IPNet",
	"macaddr":  "net.HardwareAddr",
	"macaddr8": "net.HardwareAddr",

	// Bit strings
	"bit":    "string",
	"varbit": "string",

	// Full-text search
	"tsvector": "string",
	"tsquery":  "string",

	// OID family (uint32 internally)
	"oid":           "uint32",
	"regclass":      "uint32",
	"regproc":       "uint32",
	"regprocedure":  "uint32",
	"regtype":       "uint32",
	"regrole":       "uint32",
	"regnamespace":  "uint32",
	"regconfig":     "uint32",
	"regdictionary": "uint32",

	// System scalar types that can appear in schemas
	"pg_lsn":        "string",
	"txid_snapshot": "string",

	// Geometric (string unless you build structs)
	"point":   "string",
	"line":    "string",
	"lseg":    "string",
	"box":     "string",
	"path":    "string",
	"polygon": "string",
	"circle":  "string",

	// Ranges
	"int4range": "string",
	"int8range": "string",
	"numrange":  "string",
	"tsrange":   "string",
	"tstzrange": "string",
	"daterange": "string",

	// Multiranges
	"int4multirange": "string",
	"int8multirange": "string",
	"nummultirange":  "string",
	"tsmultirange":   "string",
	"tstzmultirange": "string",
	"datemultirange": "string",

	// Money (locale-sensitive; treat as string or cents externally)
	"money": "string",
}
