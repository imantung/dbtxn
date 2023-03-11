package dbtxn

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type (
	key int
	// Context of transaction
	Context struct {
		TxMap map[*sql.DB]Tx
		Errs  []error
	}
	// CommitFn is commit function to close the transaction
	CommitFn func() error
	// UseHandler responsible to handle transaction
	UseHandler struct {
		*Context
		DB
	}
	// Tx is interface for *db.Tx
	Tx interface {
		DB
		Rollback() error
		Commit() error
	}
	// DBConn is interface for *db.DB
	DB interface {
		Query(string, ...interface{}) (*sql.Rows, error)
		QueryRow(string, ...interface{}) *sql.Row
		Exec(string, ...interface{}) (sql.Result, error)
		QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...interface{}) *sql.Row
		ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	}
)

// ContextKey to get transaction
const ContextKey key = iota

var (
	ErrSep = "; "
)

// NewContext return new instance of Context
func NewContext() *Context {
	return &Context{TxMap: make(map[*sql.DB]Tx)}
}

// Begin transaction
func Begin(parent *context.Context) *Context {
	c := NewContext()
	*parent = context.WithValue(*parent, ContextKey, c)
	return c
}

// Use transaction if possible
func Use(ctx context.Context, db *sql.DB) (*UseHandler, error) {
	if ctx == nil {
		return nil, errors.New("dbtxn: missing context.Context")
	}

	c := Find(ctx)
	if c == nil { // NOTE: not transactional
		return &UseHandler{DB: db}, nil
	}

	tx, err := c.Begin(ctx, db)
	if err != nil {
		return nil, err
	}

	return &UseHandler{DB: tx, Context: c}, nil
}

// Find transaction context
func Find(ctx context.Context) *Context {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(ContextKey).(*Context)
	return c
}

// Error of transaction
func Error(ctx context.Context) error {
	if c := Find(ctx); c != nil {
		var msgs []string
		for _, err := range c.Errs {
			if err != nil {
				msgs = append(msgs, err.Error())
			}
		}
		if errMsg := strings.Join(msgs, ErrSep); errMsg != "" {
			return errors.New(errMsg)
		}
	}
	return nil
}

//
// Context
//

// Begin transaction
func (c *Context) Begin(ctx context.Context, db *sql.DB) (DB, error) {
	tx, ok := c.TxMap[db]
	if ok {
		return tx, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		c.AppendError(err)
		return nil, err
	}
	c.TxMap[db] = tx
	return tx, nil
}

// Commit if no error
func (c *Context) Commit() error {
	var errMsgs []string
	if len(c.Errs) > 0 {
		for _, tx := range c.TxMap {
			if err := tx.Rollback(); err != nil {
				errMsgs = append(errMsgs, err.Error())
			}
		}
	} else {
		for _, tx := range c.TxMap {
			if err := tx.Commit(); err != nil {
				errMsgs = append(errMsgs, err.Error())
			}
		}
	}

	if msg := strings.Join(errMsgs, ErrSep); msg != "" {
		return errors.New(msg)
	}

	return nil
}

// AppendError to append error to txn context
func (c *Context) AppendError(err error) bool {
	if c != nil && err != nil {
		c.Errs = append(c.Errs, err)
		return true
	}
	return false
}
