// Package models holds the domain entities used by the core layer. These are
// plain structs — no GORM tags, no TableName(). The repository owns the mapping
// between these entities and the database rows (its private *Row structs and
// their ToEntity methods).
//
// The schema is managed with explicit DDL in scripts/db/schema.sql, not GORM
// auto-migration.
package models
