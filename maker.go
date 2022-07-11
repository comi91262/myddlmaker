package myddlmaker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

type Config struct {
	DB          *DBConfig
	OutFilePath string
}

type DBConfig struct {
	Driver  string
	Engine  string
	Charset string
}

type Maker struct {
	config  *Config
	structs []any
	tables  []*table
}

func New(config *Config) (*Maker, error) {
	return &Maker{
		config: config,
	}, nil
}

func (m *Maker) AddStructs(structs ...any) {
	m.structs = append(m.structs, structs...)
}

// GenerateFile opens
func (m *Maker) GenerateFile() error {
	f, err := os.Create(m.config.OutFilePath)
	if err != nil {
		return fmt.Errorf("myddlmaker: failed to open %q: %w", m.config.OutFilePath, err)
	}
	defer f.Close()

	if err := m.Generate(f); err != nil {
		return fmt.Errorf("myddlmaker: failed to generate ddl: %w", err)
	}

	return f.Close()
}

func (m *Maker) Generate(w io.Writer) error {
	var buf bytes.Buffer
	if err := m.parse(); err != nil {
		return err
	}

	buf.WriteString("SET foreign_key_checks=0;\n")
	for _, table := range m.tables {
		fmt.Fprintf(&buf, "DROP TABLE IF EXISTS %s;\n\n", quote(table.name))
		fmt.Fprintf(&buf, "CREATE TABLE %s (\n", quote(table.name))
		for _, col := range table.columns {
			fmt.Fprintf(&buf, "    %s %s,\n", quote(col.name), col.typ)
		}
		fmt.Fprintf(&buf, "    PRIMARY KEY (`id`)") // FIX ME
		fmt.Fprintf(&buf, ") ENGINE=InnoDB DEFAULT CHARACTER SET = 'utf8mb4';\n\n")
	}

	buf.WriteString("SET foreign_key_checks=1;\n")

	if _, err := buf.WriteTo(w); err != nil {
		return err
	}
	return nil
}

func (m *Maker) parse() error {
	m.tables = make([]*table, len(m.structs))
	for i, s := range m.structs {
		tbl, err := newTable(s)
		if err != nil {
			return fmt.Errorf("myddlmaker: failed to parse: %w", err)
		}
		m.tables[i] = tbl
	}
	return nil
}

func quote(s string) string {
	var buf strings.Builder
	// Strictly speaking, we need to count the number of backquotes in s.
	// However, in many cases, s doesn't include backquotes.
	buf.Grow(len(s) + len("``"))

	buf.WriteByte('`')
	for _, r := range s {
		if r == '`' {
			buf.WriteByte('`')
		}
		buf.WriteRune(r)
	}
	buf.WriteByte('`')
	return buf.String()
}
