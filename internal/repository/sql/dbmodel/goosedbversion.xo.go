package dbmodel

// Code generated by xo. DO NOT EDIT.

import (
	"context"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx postgres driver
)

// GooseDbVersion represents a row from 'public.goose_db_version'.
type GooseDbVersion struct {
	ID        int       `json:"id" db:"id"`                 // id
	VersionID int64     `json:"version_id" db:"version_id"` // version_id
	IsApplied bool      `json:"is_applied" db:"is_applied"` // is_applied
	Tstamp    time.Time `json:"tstamp" db:"tstamp"`         // tstamp
	// xo fields
	_exists, _deleted bool
}

// Exists returns true when the [GooseDbVersion] exists in the database.
func (gdv *GooseDbVersion) Exists() bool {
	return gdv._exists
}

// Deleted returns true when the [GooseDbVersion] has been marked for deletion
// from the database.
func (gdv *GooseDbVersion) Deleted() bool {
	return gdv._deleted
}

// Insert inserts the [GooseDbVersion] to the database.
func (gdv *GooseDbVersion) Insert(ctx context.Context, db DB) error {
	switch {
	case gdv._exists: // already exists
		return logerror(&ErrInsertFailed{ErrAlreadyExists})
	case gdv._deleted: // deleted
		return logerror(&ErrInsertFailed{ErrMarkedForDeletion})
	}
	// insert (primary key generated and returned by database)
	const sqlstr = `INSERT INTO public.goose_db_version (` +
		`version_id, is_applied, tstamp` +
		`) VALUES (` +
		`$1, $2, $3` +
		`) RETURNING id`
	// run
	logf(sqlstr, gdv.VersionID, gdv.IsApplied, gdv.Tstamp)
	if err := db.QueryRow(ctx, sqlstr, gdv.VersionID, gdv.IsApplied, gdv.Tstamp).Scan(&gdv.ID); err != nil {
		return logerror(err)
	}
	// set exists
	gdv._exists = true
	return nil
}

// Update updates a [GooseDbVersion] in the database.
func (gdv *GooseDbVersion) Update(ctx context.Context, db DB) error {
	switch {
	case !gdv._exists: // doesn't exist
		return logerror(&ErrUpdateFailed{ErrDoesNotExist})
	case gdv._deleted: // deleted
		return logerror(&ErrUpdateFailed{ErrMarkedForDeletion})
	}
	// update with composite primary key
	const sqlstr = `UPDATE public.goose_db_version SET ` +
		`version_id = $1, is_applied = $2, tstamp = $3 ` +
		`WHERE id = $4`
	// run
	logf(sqlstr, gdv.VersionID, gdv.IsApplied, gdv.Tstamp, gdv.ID)
	if _, err := db.Exec(ctx, sqlstr, gdv.VersionID, gdv.IsApplied, gdv.Tstamp, gdv.ID); err != nil {
		return logerror(err)
	}
	return nil
}

// Save saves the [GooseDbVersion] to the database.
func (gdv *GooseDbVersion) Save(ctx context.Context, db DB) error {
	if gdv.Exists() {
		return gdv.Update(ctx, db)
	}
	return gdv.Insert(ctx, db)
}

// Upsert performs an upsert for [GooseDbVersion].
func (gdv *GooseDbVersion) Upsert(ctx context.Context, db DB) error {
	switch {
	case gdv._deleted: // deleted
		return logerror(&ErrUpsertFailed{ErrMarkedForDeletion})
	}
	// upsert
	const sqlstr = `INSERT INTO public.goose_db_version (` +
		`id, version_id, is_applied, tstamp` +
		`) VALUES (` +
		`$1, $2, $3, $4` +
		`)` +
		` ON CONFLICT (id) DO ` +
		`UPDATE SET ` +
		`version_id = EXCLUDED.version_id, is_applied = EXCLUDED.is_applied, tstamp = EXCLUDED.tstamp `
	// run
	logf(sqlstr, gdv.ID, gdv.VersionID, gdv.IsApplied, gdv.Tstamp)
	if _, err := db.Exec(ctx, sqlstr, gdv.ID, gdv.VersionID, gdv.IsApplied, gdv.Tstamp); err != nil {
		return logerror(err)
	}
	// set exists
	gdv._exists = true
	return nil
}

// Delete deletes the [GooseDbVersion] from the database.
func (gdv *GooseDbVersion) Delete(ctx context.Context, db DB) error {
	switch {
	case !gdv._exists: // doesn't exist
		return nil
	case gdv._deleted: // deleted
		return nil
	}
	// delete with single primary key
	const sqlstr = `DELETE FROM public.goose_db_version ` +
		`WHERE id = $1`
	// run
	logf(sqlstr, gdv.ID)
	if _, err := db.Exec(ctx, sqlstr, gdv.ID); err != nil {
		return logerror(err)
	}
	// set deleted
	gdv._deleted = true
	return nil
}

// GooseDbVersionByID retrieves a row from 'public.goose_db_version' as a [GooseDbVersion].
//
// Generated from index 'goose_db_version_pkey'.
func GooseDbVersionByID(ctx context.Context, db DB, id int) (*GooseDbVersion, error) {
	// query
	const sqlstr = `SELECT ` +
		`id, version_id, is_applied, tstamp ` +
		`FROM public.goose_db_version ` +
		`WHERE id = $1`
	// run
	logf(sqlstr, id)
	gdv := GooseDbVersion{
		_exists: true,
	}
	if err := db.QueryRow(ctx, sqlstr, id).Scan(&gdv.ID, &gdv.VersionID, &gdv.IsApplied, &gdv.Tstamp); err != nil {
		return nil, logerror(err)
	}
	return &gdv, nil
}

// GooseDbVersionByIDs retrieves a row from 'public.goose_db_version' as a [GooseDbVersion].
//
// Generated from index 'goose_db_version_pkey'.
func GooseDbVersionByIDs(ctx context.Context, db DB, id []int) ([]*GooseDbVersion, error) {
	// query
	const sqlstr = `SELECT ` +
		`id, version_id, is_applied, tstamp ` +
		`FROM public.goose_db_version ` +
		`WHERE id = ANY($1) ` +
		`ORDER BY id`
	// run
	logf(sqlstr, id)

	rows, err := db.Query(ctx, sqlstr, id)
	if err != nil {
		return nil, logerror(err)
	}
	defer rows.Close()
	// process
	var res []*GooseDbVersion
	for rows.Next() {
		gdv := GooseDbVersion{
			_exists: true,
		}
		// scan
		if err := rows.Scan(&gdv.ID, &gdv.VersionID, &gdv.IsApplied, &gdv.Tstamp); err != nil {
			return nil, logerror(err)
		}
		res = append(res, &gdv)
	}
	if err := rows.Err(); err != nil {
		return nil, logerror(err)
	}
	return res, nil
}
