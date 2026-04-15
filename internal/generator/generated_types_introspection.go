package generator

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type postgresEnumMetadata struct {
	SchemaName string
	TypeName   string
	Labels     []string
}

func (m postgresEnumMetadata) canonicalName() string {
	return m.TypeName
}

func (m postgresEnumMetadata) qualifiedName() string {
	return m.SchemaName + "." + m.TypeName
}

type postgresDomainMetadata struct {
	SchemaName    string
	DomainName    string
	BaseSchema    string
	BaseTypeName  string
	ConstraintDef []string
}

func (m postgresDomainMetadata) canonicalName() string {
	return m.DomainName
}

func (m postgresDomainMetadata) qualifiedName() string {
	return m.SchemaName + "." + m.DomainName
}

func loadPostgresEnumMetadata(db *gorm.DB) (map[string]postgresEnumMetadata, error) {
	var rows []struct {
		SchemaName string `gorm:"column:schema_name"`
		TypeName   string `gorm:"column:type_name"`
		Label      string `gorm:"column:label"`
	}

	if err := db.Raw(`
		SELECT
			ns.nspname AS schema_name,
			t.typname AS type_name,
			e.enumlabel AS label
		FROM pg_type t
		JOIN pg_namespace ns ON ns.oid = t.typnamespace
		JOIN pg_enum e ON e.enumtypid = t.oid
		WHERE t.typtype = 'e'
		ORDER BY ns.nspname, t.typname, e.enumsortorder
	`).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load PostgreSQL enum metadata: %w", err)
	}

	indexed := make(map[string]*postgresEnumMetadata, len(rows))
	for _, row := range rows {
		key := row.SchemaName + "." + row.TypeName
		meta := indexed[key]
		if meta == nil {
			meta = &postgresEnumMetadata{
				SchemaName: row.SchemaName,
				TypeName:   row.TypeName,
			}
			indexed[key] = meta
		}
		meta.Labels = append(meta.Labels, row.Label)
	}

	out := make(map[string]postgresEnumMetadata, len(indexed)*2)
	for _, meta := range indexed {
		out[meta.canonicalName()] = *meta
		out[meta.qualifiedName()] = *meta
	}

	return out, nil
}

func loadPostgresDomainMetadata(db *gorm.DB) (map[string]postgresDomainMetadata, error) {
	var rows []struct {
		SchemaName    string         `gorm:"column:schema_name"`
		DomainName    string         `gorm:"column:domain_name"`
		BaseSchema    string         `gorm:"column:base_schema"`
		BaseTypeName  string         `gorm:"column:base_type_name"`
		ConstraintDef sql.NullString `gorm:"column:constraint_def"`
	}

	if err := db.Raw(`
		SELECT
			ns.nspname AS schema_name,
			t.typname AS domain_name,
			bns.nspname AS base_schema,
			b.typname AS base_type_name,
			pg_get_constraintdef(c.oid, true) AS constraint_def
		FROM pg_type t
		JOIN pg_namespace ns ON ns.oid = t.typnamespace
		JOIN pg_type b ON b.oid = t.typbasetype
		JOIN pg_namespace bns ON bns.oid = b.typnamespace
		LEFT JOIN pg_constraint c ON c.contypid = t.oid AND c.contype = 'c'
		WHERE t.typtype = 'd'
		ORDER BY ns.nspname, t.typname, c.conname
	`).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load PostgreSQL domain metadata: %w", err)
	}

	indexed := make(map[string]*postgresDomainMetadata, len(rows))
	for _, row := range rows {
		key := row.SchemaName + "." + row.DomainName
		meta := indexed[key]
		if meta == nil {
			meta = &postgresDomainMetadata{
				SchemaName:   row.SchemaName,
				DomainName:   row.DomainName,
				BaseSchema:   row.BaseSchema,
				BaseTypeName: row.BaseTypeName,
			}
			indexed[key] = meta
		}
		if row.ConstraintDef.Valid {
			constraint := strings.TrimSpace(row.ConstraintDef.String)
			if constraint != "" {
				meta.ConstraintDef = append(meta.ConstraintDef, constraint)
			}
		}
	}

	out := make(map[string]postgresDomainMetadata, len(indexed)*2)
	for _, meta := range indexed {
		out[meta.canonicalName()] = *meta
		out[meta.qualifiedName()] = *meta
	}

	return out, nil
}
