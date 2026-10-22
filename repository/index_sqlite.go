package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var _ RepoIndex = &sqliteRepoIndex{}

// sqliteRepoIndex keeps the indexes of every namespace in a single SQLite
// database. SQLite does the locking between processes: in WAL mode, readers
// see the latest committed state while a single writer works.
type sqliteRepoIndex struct {
	path string

	mu sync.Mutex
	db *sql.DB
}

// NewSqliteIndex returns the indexes in the SQLite database at path, created as
// needed, or in memory if path is ":memory:".
func NewSqliteIndex(path string) RepoIndex {
	return &sqliteRepoIndex{path: path}
}

// sqliteLayout is the version of the schema and of textTokens. Each layout has
// a file of its own, for processes of different layouts to share the
// repository without emptying the database of each other.
const sqliteLayout = 1

// sqliteSchema holds the documents of every namespace:
//
//	namespaces             docs                              fts (FTS5)
//	+-------------+        +-----------------------+         +--------+
//	| num      PK |<-------| space       FK        |         |        |
//	| ns          | 1    * | num         PK        |-------->| rowid  |
//	| version     |        | id          per space | 1  0..1 | tokens |
//	+-------------+        | built_from  commit    |         +--------+
//	                       | excerpt     JSONB     |
//	                       +-----------------------+
//
// In detail:
//   - namespaces: one row per namespace ("bugs", "identities") and version of
//     the format of its documents, made by its first write. Two versions of a
//     namespace have separate documents, for processes of different versions to
//     share the database without reading the documents of each other.
//   - docs: one row per document of a namespace and version: the id of its
//     entity, the commit it was built from, and its excerpt, a JSON object in
//     JSONB. num is only internal, linking to fts.
//   - fts: the tokens of the texts of a document, as textTokens gives them, with
//     textsSeparator between two texts. The table is contentless: the tokens
//     are indexed, not stored, and can't be read back. A document without text
//     has no row.
//
// For instance, a bug titled "Crash on login", with a comment "It happens every
// time":
//
//	namespaces  (3, "bugs", 6)
//	docs        (12, 3, "0a1b…", "3f2e…", {"Title": "Crash on login", …})
//	fts         (12, "crash login _ happen everi time")
//
// Deleting a version of a namespace deletes its documents, by the foreign key,
// and deleting or replacing a document deletes its tokens, by the triggers: a
// virtual table can't have a foreign key. SQLite reuses the num of a deleted
// document, which would otherwise inherit the tokens left behind.
const sqliteSchema = `
CREATE TABLE IF NOT EXISTS namespaces (
	num     INTEGER PRIMARY KEY,
	ns      TEXT NOT NULL,
	version INTEGER NOT NULL,
	UNIQUE (ns, version)
) STRICT;
CREATE TABLE IF NOT EXISTS docs (
	num        INTEGER PRIMARY KEY,
	space      INTEGER NOT NULL REFERENCES namespaces ON DELETE CASCADE,
	id         TEXT NOT NULL,
	built_from TEXT NOT NULL,
	excerpt    BLOB NOT NULL CHECK (json_valid(excerpt, 8) AND json_type(excerpt) = 'object'), -- JSONB
	UNIQUE (space, id)
) STRICT;
CREATE VIRTUAL TABLE IF NOT EXISTS fts USING fts5(
	tokens, content='', contentless_delete=1, tokenize="unicode61 tokenchars '_'"
);
CREATE TRIGGER IF NOT EXISTS docs_delete AFTER DELETE ON docs BEGIN
	DELETE FROM fts WHERE rowid = old.num;
END;
CREATE TRIGGER IF NOT EXISTS docs_update AFTER UPDATE ON docs BEGIN
	DELETE FROM fts WHERE rowid = old.num;
END;`

// textsSeparator separates the tokens of two texts of a document, for a phrase
// not to span them: textTokens never gives it, so no phrase has it.
const textsSeparator = "_"

func (s *sqliteRepoIndex) GetIndex(namespace string, version uint) (_ Index, err error) {
	defer func() { err = brokenSqliteError(s.path, err) }()
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		db, err := openSqlite(s.path)
		if err != nil {
			return nil, err
		}
		s.db = db
	}

	return &sqliteIndex{db: s.db, path: s.path, ns: namespace, version: version}, nil
}

// brokenSqliteError returns err, with what to do about it if it tells:
//   - the database at path is not one, or a corrupt one: the index only holds
//     what the cache can rebuild from the repository, so the file can go;
//   - the file system can't lock the database or share its memory between
//     processes, as WAL needs, such as a network file system or a folder shared
//     with a virtual machine.
func brokenSqliteError(path string, err error) error {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return err
	}
	switch sqliteErr.Code() {
	case sqlite3.SQLITE_IOERR_SHMOPEN, sqlite3.SQLITE_IOERR_SHMSIZE,
		sqlite3.SQLITE_IOERR_SHMMAP, sqlite3.SQLITE_IOERR_SHMLOCK,
		sqlite3.SQLITE_IOERR_LOCK:
		return fmt.Errorf("the file system of the index %s doesn't support the locking git-bug needs, such as a network file system or a shared folder: move the repository to a local disk: %w", path, err)
	}
	switch sqliteErr.Code() & 0xff {
	case sqlite3.SQLITE_NOTADB, sqlite3.SQLITE_CORRUPT:
		return fmt.Errorf("the index %s is damaged, delete it for git-bug to rebuild it from the repository: %w", path, err)
	}
	return err
}

// openSqlite opens the database at path, its tables made as needed.
func openSqlite(path string) (*sql.DB, error) {
	if path != ":memory:" {
		err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	// A write transaction takes the lock as it starts, for two writers to wait
	// on each other rather than fail. synchronous=NORMAL can lose the latest
	// transactions on a power loss, which a rebuild recovers.
	// The path is escaped for the URI, where ? starts the options, # a fragment,
	// and % an escape.
	escaped := strings.NewReplacer("%", "%25", "?", "%3f", "#", "%23").Replace(path)
	db, err := sql.Open("sqlite", "file:"+escaped+
		"?_txlock=immediate"+
		"&_pragma=busy_timeout(10000)"+
		"&_pragma=journal_mode(WAL)"+
		"&_pragma=synchronous(NORMAL)"+
		"&_pragma=foreign_keys(ON)")
	if err != nil {
		return nil, err
	}
	if path == ":memory:" {
		// each connection would have a database of its own
		db.SetMaxOpenConns(1)
	}

	// in a transaction, for another process to never see the tables half made
	tx, err := db.Begin()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(sqliteSchema)
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func (s *sqliteRepoIndex) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

var _ Index = &sqliteIndex{}

// sqliteIndex is the index of a namespace at a version, see sqliteRepoIndex.
type sqliteIndex struct {
	db      *sql.DB
	path    string
	ns      string
	version uint
}

func (i *sqliteIndex) NewBatch() IndexBatch {
	return &sqliteBatch{index: i, changes: make(map[string]*sqliteDoc)}
}

func (i *sqliteIndex) AllBuiltFrom(ctx context.Context) (_ map[string]Hash, err error) {
	defer func() { err = brokenSqliteError(i.path, err) }()
	rows, err := i.db.QueryContext(ctx, `
		SELECT id, built_from
		FROM docs
		WHERE space = (SELECT num FROM namespaces WHERE ns = ? AND version = ?)`,
		i.ns, i.version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	builtFrom := make(map[string]Hash)
	for rows.Next() {
		var id string
		var commit Hash
		err = rows.Scan(&id, &commit)
		if err != nil {
			return nil, err
		}
		builtFrom[id] = commit
	}
	return builtFrom, rows.Err()
}

func (i *sqliteIndex) BuiltFrom(ctx context.Context, id string) (_ Hash, err error) {
	defer func() { err = brokenSqliteError(i.path, err) }()
	var commit Hash
	err = i.db.QueryRowContext(ctx, `
		SELECT built_from
		FROM docs
		WHERE space = (SELECT num FROM namespaces WHERE ns = ? AND version = ?)
		  AND id = ?`,
		i.ns, i.version, id).Scan(&commit)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return commit, err
}

func (i *sqliteIndex) Excerpt(ctx context.Context, id string) (_ Hash, _ []byte, err error) {
	defer func() { err = brokenSqliteError(i.path, err) }()
	var commit Hash
	var excerpt []byte
	err = i.db.QueryRowContext(ctx, `
		SELECT built_from, json(excerpt)
		FROM docs
		WHERE space = (SELECT num FROM namespaces WHERE ns = ? AND version = ?)
		  AND id = ?`,
		i.ns, i.version, id).Scan(&commit, &excerpt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrNotFound
	}
	return commit, excerpt, err
}

func (i *sqliteIndex) Excerpts(ctx context.Context) (_ map[string]Hash, _ map[string][]byte, err error) {
	defer func() { err = brokenSqliteError(i.path, err) }()
	rows, err := i.db.QueryContext(ctx, `
		SELECT id, built_from, json(excerpt)
		FROM docs
		WHERE space = (SELECT num FROM namespaces WHERE ns = ? AND version = ?)`,
		i.ns, i.version)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	builtFrom := make(map[string]Hash)
	excerpts := make(map[string][]byte)
	for rows.Next() {
		var id string
		var commit Hash
		var excerpt []byte
		err = rows.Scan(&id, &commit, &excerpt)
		if err != nil {
			return nil, nil, err
		}
		builtFrom[id] = commit
		excerpts[id] = excerpt
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, err
	}
	return builtFrom, excerpts, nil
}

func (i *sqliteIndex) Search(ctx context.Context, phrases []string) (_ []string, err error) {
	defer func() { err = brokenSqliteError(i.path, err) }()
	// Each phrase is a string for FTS5, quoted, so never a query syntax. Its
	// tokens can't hold a quote.
	var match []string
	for _, phrase := range phrases {
		tokens := textTokens(phrase)
		if len(tokens) > 0 {
			match = append(match, `"`+strings.Join(tokens, " ")+`"`)
		}
	}
	if len(match) == 0 {
		return nil, nil
	}

	rows, err := i.db.QueryContext(ctx, `
		SELECT docs.id
		FROM fts
		JOIN docs ON docs.num = fts.rowid
		WHERE fts MATCH ?
		  AND docs.space = (SELECT num FROM namespaces WHERE ns = ? AND version = ?)`,
		strings.Join(match, " OR "), i.ns, i.version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type sqliteBatch struct {
	index *sqliteIndex
	// the latest change of each document: its new version, or nil if removed
	changes map[string]*sqliteDoc
}

type sqliteDoc struct {
	commit Hash
	// text, as jsonb() takes a blob for JSONB already
	excerpt string
	// the tokens of the texts, see textTokens and textsSeparator
	tokens string
}

func (b *sqliteBatch) Set(id string, commit Hash, excerpt []byte, texts []string) error {
	if !commit.IsValid() {
		return fmt.Errorf("invalid commit %q for document %s", commit, id)
	}
	// the texts are analyzed here, not in Apply which holds the lock
	var parts []string
	for _, text := range texts {
		tokens := textTokens(text)
		if len(tokens) > 0 {
			parts = append(parts, strings.Join(tokens, " "))
		}
	}
	b.changes[id] = &sqliteDoc{
		commit:  commit,
		excerpt: string(excerpt),
		tokens:  strings.Join(parts, " "+textsSeparator+" "),
	}
	return nil
}

func (b *sqliteBatch) Remove(id string) {
	b.changes[id] = nil
}

func (b *sqliteBatch) Apply(ctx context.Context) (err error) {
	defer func() { err = brokenSqliteError(b.index.path, err) }()
	tx, err := b.index.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// made by the first write of a version, unless another process did already
	_, err = tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO namespaces (ns, version)
		VALUES (?, ?)`,
		b.index.ns, b.index.version)
	if err != nil {
		return err
	}
	var space int64
	err = tx.QueryRowContext(ctx, `
		SELECT num
		FROM namespaces
		WHERE ns = ? AND version = ?`,
		b.index.ns, b.index.version).Scan(&space)
	if err != nil {
		return err
	}

	for id, doc := range b.changes {
		if doc == nil {
			_, err = tx.ExecContext(ctx, "DELETE FROM docs WHERE space = ? AND id = ?", space, id)
			if err != nil {
				return err
			}
			continue
		}

		var num int64
		err = tx.QueryRowContext(ctx, `
			INSERT INTO docs (space, id, built_from, excerpt)
			VALUES (?, ?, ?, jsonb(?))
			ON CONFLICT (space, id) DO UPDATE
			SET built_from = excluded.built_from,
			    excerpt = excluded.excerpt
			RETURNING num`,
			space, id, doc.commit, doc.excerpt).Scan(&num)
		if err != nil {
			return err
		}
		if doc.tokens != "" {
			_, err = tx.ExecContext(ctx, "INSERT INTO fts (rowid, tokens) VALUES (?, ?)", num, doc.tokens)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
