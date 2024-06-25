package dbtxn

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type (
	key int

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
	// DB is interface for *db.DB
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

	c := Get(ctx)
	if c == nil { // NOTE: not transactional
		return &UseHandler{DB: db}, nil
	}

	tx, err := c.Begin(ctx, db)
	if err != nil {
		return nil, err
	}

	return &UseHandler{DB: tx, Context: c}, nil
}

// Get transaction context from context.Context
func Get(ctx context.Context) *Context {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(ContextKey).(*Context)
	return c
}

// Error of transaction
func Error(ctx context.Context) error {
	if c := Get(ctx); c != nil {
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
