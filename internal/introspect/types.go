package introspect

import (
	"time"
)

// DatabaseSchema represents the complete schema of a database
type DatabaseSchema struct {
	Name      string
	Tables    map[string]*TableSchema
	Views     map[string]*ViewSchema
	Enums     map[string]*EnumSchema
	Functions map[string]*FunctionSchema
	Sequences map[string]*SequenceSchema
	Metadata  DatabaseMetadata
}

// DatabaseMetadata contains metadata about the database
type DatabaseMetadata struct {
	Version         string
	Encoding        string
	Collation       string
	Size            int64
	TableCount      int
	IndexCount      int
	ConstraintCount int
	InspectedAt     time.Time
}

// TableSchema represents the schema of a single table
type TableSchema struct {
	Name        string
	Schema      string
	Columns     []*ColumnSchema
	PrimaryKey  *PrimaryKeySchema
	ForeignKeys []*ForeignKeySchema
	Indexes     []*IndexSchema
	Constraints []*ConstraintSchema
	Triggers    []*TriggerSchema
	Comment     string
	RowCount    int64
	SizeBytes   int64
}

// ColumnSchema represents a column definition
type ColumnSchema struct {
	Name             string
	OrdinalPosition  int
	DataType         string
	UDTName          string
	IsNullable       bool
	DefaultValue     *string
	CharMaxLength    *int
	NumericPrecision *int
	NumericScale     *int
	IsIdentity       bool
	IsGenerated      bool
	GenerationExpr   *string
	Comment          string
}

// PrimaryKeySchema represents a primary key constraint
type PrimaryKeySchema struct {
	Name    string
	Columns []string
}

// ForeignKeySchema represents a foreign key constraint
type ForeignKeySchema struct {
	Name              string
	Columns           []string
	ReferencedTable   string
	ReferencedSchema  string
	ReferencedColumns []string
	OnDelete          string
	OnUpdate          string
}

// IndexSchema represents an index
type IndexSchema struct {
	Name       string
	Columns    []IndexColumn
	IsUnique   bool
	IsPrimary  bool
	IsPartial  bool
	Where      string
	Type       string
	TableSpace string
}

// IndexColumn represents a column in an index
type IndexColumn struct {
	Name       string
	Expression string
	Order      string
	NullsOrder string
}

// ConstraintSchema represents a table constraint
type ConstraintSchema struct {
	Name       string
	Type       string
	Definition string
	Columns    []string
}

// TriggerSchema represents a trigger
type TriggerSchema struct {
	Name       string
	Timing     string
	Events     []string
	Level      string
	Function   string
	Definition string
	IsEnabled  bool
}

// ViewSchema represents a view
type ViewSchema struct {
	Name       string
	Schema     string
	Definition string
	Columns    []*ColumnSchema
	Comment    string
}

// EnumSchema represents an enum type
type EnumSchema struct {
	Name   string
	Schema string
	Values []string
}

// FunctionSchema represents a stored function or procedure
type FunctionSchema struct {
	Name       string
	Schema     string
	Arguments  []FunctionArgument
	ReturnType string
	Language   string
	Definition string
	IsVolatile bool
}

// FunctionArgument represents a function argument
type FunctionArgument struct {
	Name     string
	DataType string
	Mode     string
	Default  *string
}

// SequenceSchema represents a sequence
type SequenceSchema struct {
	Name        string
	Schema      string
	DataType    string
	StartValue  int64
	MinValue    int64
	MaxValue    int64
	Increment   int64
	CycleOption bool
	OwnedBy     string
}

// SchemaComparison represents differences between expected and actual schema
type SchemaComparison struct {
	MissingTables    []string
	ExtraTables      []string
	ModifiedTables   map[string]TableDifferences
	MissingEnums     []string
	ExtraEnums       []string
	MissingFunctions []string
	ExtraFunctions   []string
}

// TableDifferences represents differences in a table
type TableDifferences struct {
	TableName          string
	MissingColumns     []string
	ExtraColumns       []string
	ModifiedColumns    map[string]ColumnDifferences
	MissingIndexes     []string
	ExtraIndexes       []string
	MissingConstraints []string
	ExtraConstraints   []string
}

// ColumnDifferences represents differences in a column
type ColumnDifferences struct {
	ColumnName      string
	TypeChanged     bool
	OldType         string
	NewType         string
	NullableChanged bool
	OldNullable     bool
	NewNullable     bool
	DefaultChanged  bool
	OldDefault      *string
	NewDefault      *string
}

// ExportFormat represents the format for exporting schema
type ExportFormat string

const (
	ExportFormatJSON     ExportFormat = "json"
	ExportFormatYAML     ExportFormat = "yaml"
	ExportFormatMarkdown ExportFormat = "markdown"
	ExportFormatSQL      ExportFormat = "sql"
	ExportFormatDOT      ExportFormat = "dot"
)
