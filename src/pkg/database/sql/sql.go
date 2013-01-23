// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sql provides a generic interface around SQL (or SQL-like)
// databases.

// sql包提供通用的SQL数据库（或者类SQL）接口。
package sql

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"sync"
)

var drivers = make(map[string]driver.Driver)

// Register makes a database driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.

// Register使得数据库驱动可以使用事先定义好的名字使用。
// 如果使用同样的名字注册，或者是注册的的sql驱动是空的，Register会panic。
func Register(name string, driver driver.Driver) {
	if driver == nil {
		panic("sql: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("sql: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// RawBytes is a byte slice that holds a reference to memory owned by
// the database itself. After a Scan into a RawBytes, the slice is only
// valid until the next call to Next, Scan, or Close.

// RawBytes是一个字节数组，它是由数据库自己维护的一个内存空间。
// 当一个Scan被放入到RawBytes中之后，你下次调用Next，Scan或者Close就可以获取到slice了。
type RawBytes []byte

// NullString represents a string that may be null.
// NullString implements the Scanner interface so
// it can be used as a scan destination:
//
//  var s NullString
//  err := db.QueryRow("SELECT name FROM foo WHERE id=?", id).Scan(&s)
//  ...
//  if s.Valid {
//     // use s.String
//  } else {
//     // NULL value
//  }
//

// NullString代表一个可空的string。
// NUllString实现了Scanner接口，所以它可以被当做scan的目标变量使用:
//
//  var s NullString
//  err := db.QueryRow("SELECT name FROM foo WHERE id=?", id).Scan(&s)
//  ...
//  if s.Valid {
//     // use s.String
//  } else {
//     // NULL value
//  }
//
type NullString struct {
	String string
	Valid  bool // Valid is true if String is not NULL  // 如果String不是空，则Valid为true
}

// Scan implements the Scanner interface.

// Scan实现了Scanner接口。
func (ns *NullString) Scan(value interface{}) error {
	if value == nil {
		ns.String, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return convertAssign(&ns.String, value)
}

// Value implements the driver Valuer interface.

// Value实现了driver Valuer接口。
func (ns NullString) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.String, nil
}

// NullInt64 represents an int64 that may be null.
// NullInt64 implements the Scanner interface so
// it can be used as a scan destination, similar to NullString.

// NullInt64代表了可空的int64类型。
// NullInt64实现了Scanner接口，所以它和NullString一样可以被当做scan的目标变量。
type NullInt64 struct {
	Int64 int64
	Valid bool // Valid is true if Int64 is not NULL
}

// Scan implements the Scanner interface.

// Scan实现了Scaner接口。
func (n *NullInt64) Scan(value interface{}) error {
	if value == nil {
		n.Int64, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return convertAssign(&n.Int64, value)
}

// Value implements the driver Valuer interface.

// Value实现了driver Valuer接口。
func (n NullInt64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Int64, nil
}

// NullFloat64 represents a float64 that may be null.
// NullFloat64 implements the Scanner interface so
// it can be used as a scan destination, similar to NullString.

// NullFloat64代表了可空的float64类型。
// NullFloat64实现了Scanner接口，所以它和NullString一样可以被当做scan的目标变量。
type NullFloat64 struct {
	Float64 float64
	Valid   bool // Valid is true if Float64 is not NULL  // 如果Float64非空，Valid就为true。
}

// Scan implements the Scanner interface.

// Scan实现了Scanner接口。
func (n *NullFloat64) Scan(value interface{}) error {
	if value == nil {
		n.Float64, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return convertAssign(&n.Float64, value)
}

// Value implements the driver Valuer interface.

// Value实现了driver的Valuer接口。
func (n NullFloat64) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Float64, nil
}

// NullBool represents a bool that may be null.
// NullBool implements the Scanner interface so
// it can be used as a scan destination, similar to NullString.

// NullBool代表了可空的bool类型。
// NullBool实现了Scanner接口，所以它和NullString一样可以被当做scan的目标变量。
type NullBool struct {
	Bool  bool
	Valid bool // Valid is true if Bool is not NULL  // 如果Bool非空，Valid就为true
}

// Scan implements the Scanner interface.

// Scan实现了Scanner接口。
func (n *NullBool) Scan(value interface{}) error {
	if value == nil {
		n.Bool, n.Valid = false, false
		return nil
	}
	n.Valid = true
	return convertAssign(&n.Bool, value)
}

// Value implements the driver Valuer interface.

// Value实现了driver的Valuer接口。
func (n NullBool) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Bool, nil
}

// Scanner is an interface used by Scan.

// Scanner是被Scan使用的接口。
type Scanner interface {
	// Scan assigns a value from a database driver.
	//
	// The src value will be of one of the following restricted
	// set of types:
	//
	//    int64
	//    float64
	//    bool
	//    []byte
	//    string
	//    time.Time
	//    nil - for NULL values
	//
	// An error should be returned if the value can not be stored
	// without loss of information.

	// Scan从数据库驱动中设置一个值。
	//
	// src值可以是下面限定的集中类型之一:
	//
	//    int64
	//    float64
	//    bool
	//    []byte
	//    string
	//    time.Time
	//    nil - for NULL values
	//
	// 如果数据只有通过丢失信息才能存储下来，这个方法就会返回error。
	Scan(src interface{}) error
}

// ErrNoRows is returned by Scan when QueryRow doesn't return a
// row. In such a case, QueryRow returns a placeholder *Row value that
// defers this error until a Scan.

// ErrNoRows是QueryRow的时候，当没有返回任何数据，Scan会返回的错误。
// 在这种情况下，QueryRow会返回一个*Row的标示符，直到调用Scan的时候才返回这个error。
var ErrNoRows = errors.New("sql: no rows in result set")

// DB is a database handle. It's safe for concurrent use by multiple
// goroutines.
//
// If the underlying database driver has the concept of a connection
// and per-connection session state, the sql package manages creating
// and freeing connections automatically, including maintaining a free
// pool of idle connections. If observing session state is required,
// either do not share a *DB between multiple concurrent goroutines or
// create and observe all state only within a transaction. Once
// DB.Open is called, the returned Tx is bound to a single isolated
// connection. Once Tx.Commit or Tx.Rollback is called, that
// connection is returned to DB's idle connection pool.

// DB是一个数据库处理器。它能很安全地被多个goroutines并发调用。
//
// 如果对应的数据库驱动有连接和会话状态的概念，sql包就能自动管理创建和释放连接，其中包括
// 管理一个自由连接池。如果有观察会话状态的需求的话，有两种方法。多个goroutine不共用一个
// *DB，或者在事物中创建和监控所有的状态。一旦DB.Open被调用，返回的Tx是绑定在一个独立的连接
// 上的。当Tx.Commit或者Tx.Rollback被调用，连接就会返回到DB的闲置连接池。
type DB struct {
	driver driver.Driver
	dsn    string

	mu       sync.Mutex // protects freeConn and closed // 用来保护freeConn和closed属性的
	freeConn []driver.Conn
	closed   bool
}

// Open opens a database specified by its database driver name and a
// driver-specific data source name, usually consisting of at least a
// database name and connection information.
//
// Most users will open a database via a driver-specific connection
// helper function that returns a *DB.

// Open打开一个数据库，这个数据库是由其驱动名称和驱动制定的数据源信息打开的，这个数据源信息通常
// 是由至少一个数据库名字和连接信息组成的。
//
// 多数用户通过指定的驱动连接辅助函数来打开一个数据库。打开数据库之后会返回*DB。
func Open(driverName, dataSourceName string) (*DB, error) {
	driver, ok := drivers[driverName]
	if !ok {
		return nil, fmt.Errorf("sql: unknown driver %q (forgotten import?)", driverName)
	}
	return &DB{driver: driver, dsn: dataSourceName}, nil
}

// Close closes the database, releasing any open resources.

// Close关闭数据库，释放一些使用中的资源。
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	var err error
	for _, c := range db.freeConn {
		err1 := c.Close()
		if err1 != nil {
			err = err1
		}
	}
	db.freeConn = nil
	db.closed = true
	return err
}

func (db *DB) maxIdleConns() int {
	const defaultMaxIdleConns = 2
	// TODO(bradfitz): ask driver, if supported, for its default preference
	// TODO(bradfitz): let users override?
	return defaultMaxIdleConns
}

// conn returns a newly-opened or cached driver.Conn

// conn返回新创建的，或者是缓存住的driver.Conn。
func (db *DB) conn() (driver.Conn, error) {
	db.mu.Lock()
	if db.closed {
		db.mu.Unlock()
		return nil, errors.New("sql: database is closed")
	}
	if n := len(db.freeConn); n > 0 {
		conn := db.freeConn[n-1]
		db.freeConn = db.freeConn[:n-1]
		db.mu.Unlock()
		return conn, nil
	}
	db.mu.Unlock()
	return db.driver.Open(db.dsn)
}

func (db *DB) connIfFree(wanted driver.Conn) (conn driver.Conn, ok bool) {
	db.mu.Lock()
	defer db.mu.Unlock()
	for i, conn := range db.freeConn {
		if conn != wanted {
			continue
		}
		db.freeConn[i] = db.freeConn[len(db.freeConn)-1]
		db.freeConn = db.freeConn[:len(db.freeConn)-1]
		return wanted, true
	}
	return nil, false
}

// putConnHook is a hook for testing.

// putConnHook是一个测试使用的钩子。
var putConnHook func(*DB, driver.Conn)

// putConn adds a connection to the db's free pool.
// err is optionally the last error that occurred on this connection.

// putConn将连接加入到数据库的空置池中。
// error是连接过程中最后遇到的错误。
func (db *DB) putConn(c driver.Conn, err error) {
	if err == driver.ErrBadConn {
		// Don't reuse bad connections.
		return
	}
	db.mu.Lock()
	if putConnHook != nil {
		putConnHook(db, c)
	}
	if n := len(db.freeConn); !db.closed && n < db.maxIdleConns() {
		db.freeConn = append(db.freeConn, c)
		db.mu.Unlock()
		return
	}
	// TODO: check to see if we need this Conn for any prepared
	// statements which are still active?
	db.mu.Unlock()
	c.Close()
}

// Prepare creates a prepared statement for later execution.

// Prepare为后面的执行操作事先定义了声明。
func (db *DB) Prepare(query string) (*Stmt, error) {
	var stmt *Stmt
	var err error
	for i := 0; i < 10; i++ {
		stmt, err = db.prepare(query)
		if err != driver.ErrBadConn {
			break
		}
	}
	return stmt, err
}

func (db *DB) prepare(query string) (stmt *Stmt, err error) {
	// TODO: check if db.driver supports an optional
	// driver.Preparer interface and call that instead, if so,
	// otherwise we make a prepared statement that's bound
	// to a connection, and to execute this prepared statement
	// we either need to use this connection (if it's free), else
	// get a new connection + re-prepare + execute on that one.
	ci, err := db.conn()
	if err != nil {
		return nil, err
	}
	defer func() {
		db.putConn(ci, err)
	}()

	si, err := ci.Prepare(query)
	if err != nil {
		return nil, err
	}
	stmt = &Stmt{
		db:    db,
		query: query,
		css:   []connStmt{{ci, si}},
	}
	return stmt, nil
}

// Exec executes a query without returning any rows.

// Exec执行query操作，而没有返回任何行。
func (db *DB) Exec(query string, args ...interface{}) (Result, error) {
	var res Result
	var err error
	for i := 0; i < 10; i++ {
		res, err = db.exec(query, args)
		if err != driver.ErrBadConn {
			break
		}
	}
	return res, err
}

func (db *DB) exec(query string, args []interface{}) (res Result, err error) {
	ci, err := db.conn()
	if err != nil {
		return nil, err
	}
	defer func() {
		db.putConn(ci, err)
	}()

	if execer, ok := ci.(driver.Execer); ok {
		dargs, err := driverArgs(nil, args)
		if err != nil {
			return nil, err
		}
		resi, err := execer.Exec(query, dargs)
		if err != driver.ErrSkip {
			if err != nil {
				return nil, err
			}
			return result{resi}, nil
		}
	}

	sti, err := ci.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer sti.Close()

	return resultFromStatement(sti, args...)
}

// Query executes a query that returns rows, typically a SELECT.
// The args are for any placeholder parameters in the query.

// Query执行了一个有返回行的查询操作，比如SELECT。
// args 形参为该查询中的任何占位符。
func (db *DB) Query(query string, args ...interface{}) (*Rows, error) {
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(args...)
	if err != nil {
		stmt.Close()
		return nil, err
	}
	rows.closeStmt = stmt
	return rows, nil
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always return a non-nil value. Errors are deferred until
// Row's Scan method is called.

// QueryRow执行一个至多只返回一行记录的查询操作。
// QueryRow总是返回一个非空值。Error只会在调用行的Scan方法的时候才返回。
func (db *DB) QueryRow(query string, args ...interface{}) *Row {
	rows, err := db.Query(query, args...)
	return &Row{rows: rows, err: err}
}

// Begin starts a transaction. The isolation level is dependent on
// the driver.

// Begin开始一个事务。事务的隔离级别是由驱动决定的。
func (db *DB) Begin() (*Tx, error) {
	var tx *Tx
	var err error
	for i := 0; i < 10; i++ {
		tx, err = db.begin()
		if err != driver.ErrBadConn {
			break
		}
	}
	return tx, err
}

func (db *DB) begin() (tx *Tx, err error) {
	ci, err := db.conn()
	if err != nil {
		return nil, err
	}
	txi, err := ci.Begin()
	if err != nil {
		db.putConn(ci, err)
		return nil, err
	}
	return &Tx{
		db:  db,
		ci:  ci,
		txi: txi,
	}, nil
}

// Driver returns the database's underlying driver.

// Driver返回了数据库的底层驱动。
func (db *DB) Driver() driver.Driver {
	return db.driver
}

// Tx is an in-progress database transaction.
//
// A transaction must end with a call to Commit or Rollback.
//
// After a call to Commit or Rollback, all operations on the
// transaction fail with ErrTxDone.

// Tx代表运行中的数据库事务。
//
// 必须调用Commit或者Rollback来结束事务。
//
// 在调用Commit或者Rollback之后，所有后续对事务的操作就会返回ErrTxDone。
type Tx struct {
	db *DB

	// ci is owned exclusively until Commit or Rollback, at which point
	// it's returned with putConn.

	// ci会一直有值，直到Commit或者Rollback被调用以后。在释放ci的时候，它会被putConn调用返回。
	ci  driver.Conn
	txi driver.Tx

	// cimu is held while somebody is using ci (between grabConn
	// and releaseConn)

	// 当某人使用ci的时候，cimu就会被持有了（在grabConn之后releaseConn之前的时间段内）
	cimu sync.Mutex

	// done transitions from false to true exactly once, on Commit
	// or Rollback. once done, all operations fail with
	// ErrTxDone.

	// 一旦Commit或者Rollback，done这个事务标示就会从false值置为true。
	// 一旦这个标志位设置为true，所有事务的操作都会失败并返回ErrTxDone。
	done bool
}

var ErrTxDone = errors.New("sql: Transaction has already been committed or rolled back")

func (tx *Tx) close() {
	if tx.done {
		panic("double close") // internal error
	}
	tx.done = true
	tx.db.putConn(tx.ci, nil)
	tx.ci = nil
	tx.txi = nil
}

func (tx *Tx) grabConn() (driver.Conn, error) {
	if tx.done {
		return nil, ErrTxDone
	}
	tx.cimu.Lock()
	return tx.ci, nil
}

func (tx *Tx) releaseConn() {
	tx.cimu.Unlock()
}

// Commit commits the transaction.

// Commit提交事务。
func (tx *Tx) Commit() error {
	if tx.done {
		return ErrTxDone
	}
	defer tx.close()
	return tx.txi.Commit()
}

// Rollback aborts the transaction.

// Rollback回滚事务。
func (tx *Tx) Rollback() error {
	if tx.done {
		return ErrTxDone
	}
	defer tx.close()
	return tx.txi.Rollback()
}

// Prepare creates a prepared statement for use within a transaction.
//
// The returned statement operates within the transaction and can no longer
// be used once the transaction has been committed or rolled back.
//
// To use an existing prepared statement on this transaction, see Tx.Stmt.

// Prepare在一个事务中定义了一个操作的声明。
//
// 这里定义的声明操作一旦事务被调用了commited或者rollback之后就不能使用了。
//
// 关于如何使用定义好的操作声明，请参考Tx.Stmt。
func (tx *Tx) Prepare(query string) (*Stmt, error) {
	// TODO(bradfitz): We could be more efficient here and either
	// provide a method to take an existing Stmt (created on
	// perhaps a different Conn), and re-create it on this Conn if
	// necessary. Or, better: keep a map in DB of query string to
	// Stmts, and have Stmt.Execute do the right thing and
	// re-prepare if the Conn in use doesn't have that prepared
	// statement.  But we'll want to avoid caching the statement
	// in the case where we only call conn.Prepare implicitly
	// (such as in db.Exec or tx.Exec), but the caller package
	// can't be holding a reference to the returned statement.
	// Perhaps just looking at the reference count (by noting
	// Stmt.Close) would be enough. We might also want a finalizer
	// on Stmt to drop the reference count.
	ci, err := tx.grabConn()
	if err != nil {
		return nil, err
	}
	defer tx.releaseConn()

	si, err := ci.Prepare(query)
	if err != nil {
		return nil, err
	}

	stmt := &Stmt{
		db:    tx.db,
		tx:    tx,
		txsi:  si,
		query: query,
	}
	return stmt, nil
}

// Stmt returns a transaction-specific prepared statement from
// an existing statement.
//
// Example:
//  updateMoney, err := db.Prepare("UPDATE balance SET money=money+? WHERE id=?")
//  ...
//  tx, err := db.Begin()
//  ...
//  res, err := tx.Stmt(updateMoney).Exec(123.45, 98293203)

// Stmt从一个已有的声明中返回指定事务的声明。
//
// 例子:
//  updateMoney, err := db.Prepare("UPDATE balance SET money=money+? WHERE id=?")
//  ...
//  tx, err := db.Begin()
//  ...
//  res, err := tx.Stmt(updateMoney).Exec(123.45, 98293203)
func (tx *Tx) Stmt(stmt *Stmt) *Stmt {
	// TODO(bradfitz): optimize this. Currently this re-prepares
	// each time.  This is fine for now to illustrate the API but
	// we should really cache already-prepared statements
	// per-Conn. See also the big comment in Tx.Prepare.

	if tx.db != stmt.db {
		return &Stmt{stickyErr: errors.New("sql: Tx.Stmt: statement from different database used")}
	}
	ci, err := tx.grabConn()
	if err != nil {
		return &Stmt{stickyErr: err}
	}
	defer tx.releaseConn()
	si, err := ci.Prepare(stmt.query)
	return &Stmt{
		db:        tx.db,
		tx:        tx,
		txsi:      si,
		query:     stmt.query,
		stickyErr: err,
	}
}

// Exec executes a query that doesn't return rows.
// For example: an INSERT and UPDATE.

// Exec执行不返回任何行的操作。
// 例如：INSERT和UPDATE操作。
func (tx *Tx) Exec(query string, args ...interface{}) (Result, error) {
	ci, err := tx.grabConn()
	if err != nil {
		return nil, err
	}
	defer tx.releaseConn()

	if execer, ok := ci.(driver.Execer); ok {
		dargs, err := driverArgs(nil, args)
		if err != nil {
			return nil, err
		}
		resi, err := execer.Exec(query, dargs)
		if err == nil {
			return result{resi}, nil
		}
		if err != driver.ErrSkip {
			return nil, err
		}
	}

	sti, err := ci.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer sti.Close()

	return resultFromStatement(sti, args...)
}

// Query executes a query that returns rows, typically a SELECT.

// Query执行哪些返回行的查询操作，比如SELECT。
func (tx *Tx) Query(query string, args ...interface{}) (*Rows, error) {
	if tx.done {
		return nil, ErrTxDone
	}
	stmt, err := tx.Prepare(query)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(args...)
	if err != nil {
		stmt.Close()
		return nil, err
	}
	rows.closeStmt = stmt
	return rows, err
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always return a non-nil value. Errors are deferred until
// Row's Scan method is called.

// QueryRow执行的查询至多返回一行数据。
// QueryRow总是返回非空值。只有当执行行的Scan方法的时候，才会返回Error。
func (tx *Tx) QueryRow(query string, args ...interface{}) *Row {
	rows, err := tx.Query(query, args...)
	return &Row{rows: rows, err: err}
}

// connStmt is a prepared statement on a particular connection.

// connStmt代表在某个连接上定义好的声明。
type connStmt struct {
	ci driver.Conn
	si driver.Stmt
}

// Stmt is a prepared statement. Stmt is safe for concurrent use by multiple goroutines.

// Stmt是定义好的声明。多个goroutine并发使用Stmt是安全的。
type Stmt struct {
	// Immutable:

	// 不变的数据：
	db        *DB    // where we came from	// 数据从哪里来
	query     string // that created the Stmt	// 什么样的查询建立了这个Stmt
	stickyErr error  // if non-nil, this error is returned for all operations  // 如果是非空的话，所有操作都会返回这个错误。

	// If in a transaction, else both nil:

	// 只有在事务中，者两个值才都非空，其他情况下都是空的：
	tx   *Tx
	txsi driver.Stmt

	mu     sync.Mutex // protects the rest of the fields // 保护其他字段
	closed bool

	// css is a list of underlying driver statement interfaces
	// that are valid on particular connections.  This is only
	// used if tx == nil and one is found that has idle
	// connections.  If tx != nil, txsi is always used.

	// css是一个底层驱动的声明接口的数组，它只对特定的连接有效。只有当tx == nil的时候才使用，
	// 它是从在空闲连接池中获取的。如果tx != nil，就会使用txsi。
	css []connStmt
}

// Exec executes a prepared statement with the given arguments and
// returns a Result summarizing the effect of the statement.

// Exec根据给出的参数执行定义好的声明，并返回Result来显示执行的结果。
func (s *Stmt) Exec(args ...interface{}) (Result, error) {
	_, releaseConn, si, err := s.connStmt()
	if err != nil {
		return nil, err
	}
	defer releaseConn(nil)

	return resultFromStatement(si, args...)
}

func resultFromStatement(si driver.Stmt, args ...interface{}) (Result, error) {
	// -1 means the driver doesn't know how to count the number of
	// placeholders, so we won't sanity check input here and instead let the
	// driver deal with errors.

	// -1意味着驱动不知道如何计算占位符的数量，所以在这里，我们并不检查输入，而是让驱动自己来处理错误。
	if want := si.NumInput(); want != -1 && len(args) != want {
		return nil, fmt.Errorf("sql: expected %d arguments, got %d", want, len(args))
	}

	dargs, err := driverArgs(si, args)
	if err != nil {
		return nil, err
	}

	resi, err := si.Exec(dargs)
	if err != nil {
		return nil, err
	}
	return result{resi}, nil
}

// connStmt returns a free driver connection on which to execute the
// statement, a function to call to release the connection, and a
// statement bound to that connection.

// connStmt返回空闲的驱动连接，这个连接是用来执行这个声明的，并且同时定义一个函数来释放连接，
// 定义一个声明绑定连接。
func (s *Stmt) connStmt() (ci driver.Conn, releaseConn func(error), si driver.Stmt, err error) {
	if err = s.stickyErr; err != nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		err = errors.New("sql: statement is closed")
		return
	}

	// In a transaction, we always use the connection that the
	// transaction was created on.

	// 在事务中，我们总是使用事务创建的连接。
	if s.tx != nil {
		s.mu.Unlock()
		ci, err = s.tx.grabConn() // blocks, waiting for the connection. // 阻塞，等待连接。
		if err != nil {
			return
		}
		releaseConn = func(error) { s.tx.releaseConn() }
		return ci, releaseConn, s.txsi, nil
	}

	var cs connStmt
	match := false
	for _, v := range s.css {
		// TODO(bradfitz): lazily clean up entries in this
		// list with dead conns while enumerating
		if _, match = s.db.connIfFree(v.ci); match {
			cs = v
			break
		}
	}
	s.mu.Unlock()

	// Make a new conn if all are busy.
	// TODO(bradfitz): or wait for one? make configurable later?
	if !match {
		for i := 0; ; i++ {
			ci, err := s.db.conn()
			if err != nil {
				return nil, nil, nil, err
			}
			si, err := ci.Prepare(s.query)
			if err == driver.ErrBadConn && i < 10 {
				continue
			}
			if err != nil {
				return nil, nil, nil, err
			}
			s.mu.Lock()
			cs = connStmt{ci, si}
			s.css = append(s.css, cs)
			s.mu.Unlock()
			break
		}
	}

	conn := cs.ci
	releaseConn = func(err error) { s.db.putConn(conn, err) }
	return conn, releaseConn, cs.si, nil
}

// Query executes a prepared query statement with the given arguments
// and returns the query results as a *Rows.

// Query根据传递的参数执行一个声明的查询操作，然后以*Rows的结果返回查询结果。
func (s *Stmt) Query(args ...interface{}) (*Rows, error) {
	ci, releaseConn, si, err := s.connStmt()
	if err != nil {
		return nil, err
	}

	// -1 means the driver doesn't know how to count the number of
	// placeholders, so we won't sanity check input here and instead let the
	// driver deal with errors.
	if want := si.NumInput(); want != -1 && len(args) != want {
		return nil, fmt.Errorf("sql: statement expects %d inputs; got %d", si.NumInput(), len(args))
	}

	dargs, err := driverArgs(si, args)
	if err != nil {
		return nil, err
	}

	rowsi, err := si.Query(dargs)
	if err != nil {
		releaseConn(err)
		return nil, err
	}
	// Note: ownership of ci passes to the *Rows, to be freed
	// with releaseConn.
	rows := &Rows{
		db:          s.db,
		ci:          ci,
		releaseConn: releaseConn,
		rowsi:       rowsi,
	}
	return rows, nil
}

// QueryRow executes a prepared query statement with the given arguments.
// If an error occurs during the execution of the statement, that error will
// be returned by a call to Scan on the returned *Row, which is always non-nil.
// If the query selects no rows, the *Row's Scan will return ErrNoRows.
// Otherwise, the *Row's Scan scans the first selected row and discards
// the rest.
//
// Example usage:
//
//  var name string
//  err := nameByUseridStmt.QueryRow(id).Scan(&name)

// QueryRow根据传递的参数执行一个声明的查询操作。如果在执行声明过程中发生了错误，
// 这个error就会在Scan返回的*Row的时候返回，而这个*Row永远不会是nil。
// 如果查询没有任何行数据，*Row的Scan操作就会返回ErrNoRows。
// 否则，*Rows的Scan操作就会返回第一行数据，并且忽略其他行。
//
// Example usage:
//
//  var name string
//  err := nameByUseridStmt.QueryRow(id).Scan(&name)
func (s *Stmt) QueryRow(args ...interface{}) *Row {
	rows, err := s.Query(args...)
	if err != nil {
		return &Row{err: err}
	}
	return &Row{rows: rows}
}

// Close closes the statement.

// 关闭声明。
func (s *Stmt) Close() error {
	if s.stickyErr != nil {
		return s.stickyErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true

	if s.tx != nil {
		s.txsi.Close()
	} else {
		for _, v := range s.css {
			if ci, match := s.db.connIfFree(v.ci); match {
				v.si.Close()
				s.db.putConn(ci, nil)
			} else {
				// TODO(bradfitz): care that we can't close
				// this statement because the statement's
				// connection is in use?
			}
		}
	}
	return nil
}

// Rows is the result of a query. Its cursor starts before the first row
// of the result set. Use Next to advance through the rows:
//
//     rows, err := db.Query("SELECT ...")
//     ...
//     for rows.Next() {
//         var id int
//         var name string
//         err = rows.Scan(&id, &name)
//         ...
//     }
//     err = rows.Err() // get any error encountered during iteration
//     ...

// Rows代表查询的结果。它的指针最初指向结果集的第一行数据，需要使用Next来进一步操作。
//
//     rows, err := db.Query("SELECT ...")
//     ...
//     for rows.Next() {
//         var id int
//         var name string
//         err = rows.Scan(&id, &name)
//         ...
//     }
//     err = rows.Err() // get any error encountered during iteration
//     ...
type Rows struct {
	db          *DB
	ci          driver.Conn // owned; must call putconn when closed to release // 已经存在的连接；当释放连接的时候必须调用putconn
	releaseConn func(error)
	rowsi       driver.Rows

	closed    bool
	lastcols  []driver.Value
	lasterr   error
	closeStmt *Stmt // if non-nil, statement to Close on close  // 如果非空，这些声明会在close调用的时候关闭。
}

// Next prepares the next result row for reading with the Scan method.
// It returns true on success, false if there is no next result row.
// Every call to Scan, even the first one, must be preceded by a call
// to Next.

// Next获取下一行的数据以便给Scan调用。
// 在成功的时候返回true，在没有下一行数据的时候返回false。
// 每次调用来Scan获取数据，甚至是第一行数据，都需要调用Next来处理。
func (rs *Rows) Next() bool {
	if rs.closed {
		return false
	}
	if rs.lasterr != nil {
		return false
	}
	if rs.lastcols == nil {
		rs.lastcols = make([]driver.Value, len(rs.rowsi.Columns()))
	}
	rs.lasterr = rs.rowsi.Next(rs.lastcols)
	if rs.lasterr == io.EOF {
		rs.Close()
	}
	return rs.lasterr == nil
}

// Err returns the error, if any, that was encountered during iteration.

// Err返回错误。如果有错误的话，就会在循环过程中捕获到。
func (rs *Rows) Err() error {
	if rs.lasterr == io.EOF {
		return nil
	}
	return rs.lasterr
}

// Columns returns the column names.
// Columns returns an error if the rows are closed, or if the rows
// are from QueryRow and there was a deferred error.

// Columns返回列名字。
// 当rows设置了closed，Columns方法会返回error。
func (rs *Rows) Columns() ([]string, error) {
	if rs.closed {
		return nil, errors.New("sql: Rows are closed")
	}
	if rs.rowsi == nil {
		return nil, errors.New("sql: no Rows available")
	}
	return rs.rowsi.Columns(), nil
}

// Scan copies the columns in the current row into the values pointed
// at by dest.
//
// If an argument has type *[]byte, Scan saves in that argument a copy
// of the corresponding data. The copy is owned by the caller and can
// be modified and held indefinitely. The copy can be avoided by using
// an argument of type *RawBytes instead; see the documentation for
// RawBytes for restrictions on its use.
//
// If an argument has type *interface{}, Scan copies the value
// provided by the underlying driver without conversion. If the value
// is of type []byte, a copy is made and the caller owns the result.

// Scan将当前行的列输出到dest指向的目标值中。
//
// 如果有个参数是*[]byte的类型，Scan在这个参数里面存放的是相关数据的拷贝。
// 这个拷贝是调用函数的人所拥有的，并且可以随时被修改和存取。这个拷贝能避免使用*RawBytes；
// 关于这个类型的使用限制请参考文档。
//
// 如果有个参数是*interface{}类型，Scan会将底层驱动提供的这个值不做任何转换直接拷贝返回。
// 如果值是[]byte类型，Scan就会返回一份拷贝，并且调用者获得返回结果。
func (rs *Rows) Scan(dest ...interface{}) error {
	if rs.closed {
		return errors.New("sql: Rows closed")
	}
	if rs.lasterr != nil {
		return rs.lasterr
	}
	if rs.lastcols == nil {
		return errors.New("sql: Scan called without calling Next")
	}
	if len(dest) != len(rs.lastcols) {
		return fmt.Errorf("sql: expected %d destination arguments in Scan, not %d", len(rs.lastcols), len(dest))
	}
	for i, sv := range rs.lastcols {
		err := convertAssign(dest[i], sv)
		if err != nil {
			return fmt.Errorf("sql: Scan error on column index %d: %v", i, err)
		}
	}
	for _, dp := range dest {
		b, ok := dp.(*[]byte)
		if !ok {
			continue
		}
		if *b == nil {
			// If the []byte is now nil (for a NULL value),
			// don't fall through to below which would
			// turn it into a non-nil 0-length byte slice
			continue
		}
		if _, ok = dp.(*RawBytes); ok {
			continue
		}
		clone := make([]byte, len(*b))
		copy(clone, *b)
		*b = clone
	}
	return nil
}

// Close closes the Rows, preventing further enumeration. If the
// end is encountered, the Rows are closed automatically. Close
// is idempotent.

// Close关闭Rows，就禁止了进一步的枚举使用。如果遍历过程结束了，Rows就会自动关闭了。
// 关闭是非常重要的。
func (rs *Rows) Close() error {
	if rs.closed {
		return nil
	}
	rs.closed = true
	err := rs.rowsi.Close()
	rs.releaseConn(err)
	if rs.closeStmt != nil {
		rs.closeStmt.Close()
	}
	return err
}

// Row is the result of calling QueryRow to select a single row.

// Row是调用QueryRow的结果，代表了查询操作的一行数据。
type Row struct {
	// One of these two will be non-nil:

	// 这两个中的一个必须是非空：
	err  error // deferred error for easy chaining  // 将error保存从而延迟返回，这样能保证Row链表的简易实现
	rows *Rows
}

// Scan copies the columns from the matched row into the values
// pointed at by dest.  If more than one row matches the query,
// Scan uses the first row and discards the rest.  If no row matches
// the query, Scan returns ErrNoRows.

// Scan将符合的行的对应列拷贝到dest指的对应值中。
// 如果多于一个的行满足查询条件，Scan使用第一行，而忽略其他行。
// 如果没有行满足查询条件，Scan返回ErrNoRows。
func (r *Row) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}

	// TODO(bradfitz): for now we need to defensively clone all
	// []byte that the driver returned (not permitting
	// *RawBytes in Rows.Scan), since we're about to close
	// the Rows in our defer, when we return from this function.
	// the contract with the driver.Next(...) interface is that it
	// can return slices into read-only temporary memory that's
	// only valid until the next Scan/Close.  But the TODO is that
	// for a lot of drivers, this copy will be unnecessary.  We
	// should provide an optional interface for drivers to
	// implement to say, "don't worry, the []bytes that I return
	// from Next will not be modified again." (for instance, if
	// they were obtained from the network anyway) But for now we
	// don't care.
	for _, dp := range dest {
		if _, ok := dp.(*RawBytes); ok {
			return errors.New("sql: RawBytes isn't allowed on Row.Scan")
		}
	}

	defer r.rows.Close()
	if !r.rows.Next() {
		return ErrNoRows
	}
	err := r.rows.Scan(dest...)
	if err != nil {
		return err
	}

	return nil
}

// A Result summarizes an executed SQL command.

// 一个Result结构代表了一个执行过的SQL命令。
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

type result struct {
	driver.Result
}
