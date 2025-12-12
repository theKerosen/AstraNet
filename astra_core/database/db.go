package database

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/base64"
	"io"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

func NewDB(connStr string) (*DB, error) {
	conn, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS app_state (
		app_id INTEGER PRIMARY KEY,
		change_number TEXT,
		build_id TEXT,
		app_info_json TEXT,
		raw_vdf TEXT,
		last_diff_gz TEXT,
		last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.conn.Exec(query)
	return err
}

func (db *DB) GetAppState(appID int) (string, string, string, string, error) {
	var changeNumber, buildID, appInfoJSON, rawVDF string
	query := `SELECT change_number, build_id, app_info_json, raw_vdf FROM app_state WHERE app_id = ?`
	err := db.conn.QueryRow(query, appID).Scan(&changeNumber, &buildID, &appInfoJSON, &rawVDF)
	if err == sql.ErrNoRows {
		return "", "", "", "", nil
	}
	return changeNumber, buildID, appInfoJSON, rawVDF, err
}

func (db *DB) UpdateAppState(appID int, changeNumber, buildID, appInfoJSON, rawVDF string) error {
	query := `
	INSERT INTO app_state (app_id, change_number, build_id, app_info_json, raw_vdf, last_updated)
	VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(app_id) DO UPDATE
	SET change_number = excluded.change_number,
		build_id = excluded.build_id,
		app_info_json = excluded.app_info_json,
		raw_vdf = excluded.raw_vdf,
		last_updated = CURRENT_TIMESTAMP;
	`
	_, err := db.conn.Exec(query, appID, changeNumber, buildID, appInfoJSON, rawVDF)
	return err
}

func (db *DB) SaveLastDiff(appID int, diffJSON []byte) error {
	compressed, err := compressGzip(diffJSON)
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(compressed)

	query := `UPDATE app_state SET last_diff_gz = ? WHERE app_id = ?`
	_, err = db.conn.Exec(query, encoded, appID)
	return err
}

func (db *DB) GetLastDiff(appID int) ([]byte, error) {
	var encoded sql.NullString
	query := `SELECT last_diff_gz FROM app_state WHERE app_id = ?`
	err := db.conn.QueryRow(query, appID).Scan(&encoded)
	if err == sql.ErrNoRows || !encoded.Valid || encoded.String == "" {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	compressed, err := base64.StdEncoding.DecodeString(encoded.String)
	if err != nil {
		return nil, err
	}

	return decompressGzip(compressed)
}

func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
