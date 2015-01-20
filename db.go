package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

func setupDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "mars.sqlite")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var name string
	var version int
	err = db.QueryRow("select name from sqlite_master where type='table' and name='schema_versions'").Scan(&name)
	switch {
	case err == sql.ErrNoRows:
		break
	case err != nil:
		return nil, err
	default:
		err = db.QueryRow("select version from schema_versions").Scan(&version)
		if err != nil {
			return nil, err
		}
		if version != 1 {
			return nil, fmt.Errorf("existing schema is version %v, expected version 1", version)
		}
		return db, nil
	}

	err = loadSchema(db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func loadSchema(db *sql.DB) error {
	schema := `
	begin;

	create table schema_versions (version integer not null primary key);
	insert into schema_versions values (1);

	create table signin_requests (
		created_at integer not null,
		signin_id text not null,
		signin_secret text not null,
		pubkey blob not null
	);

	create table sessions (
		user_id text not null,
		last_active integer not null,
		session_id text not null,
		session_secret text not null,
		csrf_token text not null
	);

	create table users (
		user_id integer primary key autoincrement,
		pubkey blob not null,
		pin_updated_at integer,
		lat real,
		lon real
	);

	commit;`

	_, err := db.Exec(schema)
	return err
}
