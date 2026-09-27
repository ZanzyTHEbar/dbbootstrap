// Package db owns database/sql lifecycle operations without abstracting SQL.
//
// SQLite is available through SQLite. The libsql and go-libsql implementations
// live in the libsql and golibsql subpackages. Choose one libSQL implementation
// per executable because both upstream packages register the database/sql
// driver name "libsql".
package db
