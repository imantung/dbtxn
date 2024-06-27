package dbtxn

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"go.uber.org/multierr"
)

type (
	// Context of transaction
	Context struct {
		TxMap map[*sql.DB]Tx
		Errs  []error
	}
)

// NewContext return new instance of Context
func NewContext() *Context {
	return &Context{TxMap: make(map[*sql.DB]Tx)}
}

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

func (c *Context) CommitWithError(err *error) {
	*err = multierr.Append(*err, c.Commit())
}

// AppendError to append error to txn context
func (c *Context) AppendError(err error) bool {
	if c != nil && err != nil {
		c.Errs = append(c.Errs, err)
		return true
	}
	return false
}
