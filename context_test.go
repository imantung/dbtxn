package dbtxn_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/imantung/dbtxn"
	"github.com/stretchr/testify/require"
)

func TestContext_Commit(t *testing.T) {
	t.Run("expect rollback when error", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		mock.ExpectBegin()
		mock.ExpectRollback()

		c := dbtxn.NewContext()
		c.Begin(context.Background(), db)
		c.AppendError(errors.New("some-error"))

		require.NoError(t, c.Commit())
	})
	t.Run("expect error rollback ", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		mock.ExpectBegin()
		mock.ExpectRollback().WillReturnError(errors.New("rollback-error"))

		c := dbtxn.NewContext()
		c.Begin(context.Background(), db)
		c.AppendError(errors.New("some-error"))

		require.EqualError(t, c.Commit(), "rollback-error")
	})
	t.Run("expect commit when no error", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		mock.ExpectBegin()
		mock.ExpectCommit()

		c := dbtxn.NewContext()
		c.Begin(context.Background(), db)
		require.NoError(t, c.Commit())
	})

	t.Run("expect commit when no error", func(t *testing.T) {
		db, mock, _ := sqlmock.New()
		mock.ExpectBegin()
		mock.ExpectCommit().WillReturnError(errors.New("commit-error"))

		c := dbtxn.NewContext()
		c.Begin(context.Background(), db)
		require.EqualError(t, c.Commit(), "commit-error")
	})
}

func TestContext_CommitWithError(t *testing.T) {
	t.Run("return rollback error when no function error", func(t *testing.T) {
		fn := func(ctx context.Context) (err error) {
			db, mock, _ := sqlmock.New()
			mock.ExpectBegin()
			mock.ExpectRollback().WillReturnError(errors.New("rollback-error"))

			c := dbtxn.NewContext()
			c.Begin(ctx, db)
			c.AppendError(errors.New("error-to-trigger-rollback"))
			defer c.CommitWithError(&err)
			return nil
		}

		err := fn(context.Background())
		require.EqualError(t, err, "rollback-error")
	})

	t.Run("return function error although rollback-error", func(t *testing.T) {
		fn := func(ctx context.Context) (err error) {
			db, mock, _ := sqlmock.New()
			mock.ExpectBegin()
			mock.ExpectRollback().WillReturnError(errors.New("rollback-error"))

			c := dbtxn.NewContext()
			c.Begin(ctx, db)
			c.AppendError(errors.New("error-to-trigger-rollback"))
			defer c.CommitWithError(&err)
			return errors.New("function-error")
		}

		err := fn(context.Background())
		require.EqualError(t, err, "function-error")
	})

}
