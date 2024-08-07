package dbtxn_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/imantung/dbtxn"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	testcases := []struct {
		TestName        string
		Ctx             context.Context
		ExpectedContext *dbtxn.Context
	}{
		{
			Ctx:             nil,
			ExpectedContext: nil,
		},
		{
			Ctx:             context.Background(),
			ExpectedContext: nil,
		},
		{
			Ctx:             context.WithValue(context.Background(), dbtxn.ContextKey, "meh"),
			ExpectedContext: nil,
		},
		{
			Ctx:             context.WithValue(context.Background(), dbtxn.ContextKey, &dbtxn.Context{}),
			ExpectedContext: &dbtxn.Context{},
		},
	}
	for _, tt := range testcases {
		t.Run(tt.TestName, func(t *testing.T) {
			require.Equal(t, tt.ExpectedContext, dbtxn.Get(tt.Ctx))
		})
	}
}

func TestUse(t *testing.T) {
	testcases := []struct {
		TestName    string
		Ctx         context.Context
		DB          *sql.DB
		Expected    *dbtxn.UseHandler
		ExpectedErr string
	}{
		{
			Ctx:         nil,
			ExpectedErr: "dbtxn: missing context.Context",
		},
		{
			TestName: "non transactional",
			DB:       &sql.DB{},
			Ctx:      context.Background(),
			Expected: &dbtxn.UseHandler{DB: &sql.DB{}},
		},
		{
			TestName: "begin error",
			DB: func() *sql.DB {
				db, mock, _ := sqlmock.New()
				mock.ExpectBegin().WillReturnError(errors.New("begin-error"))
				return db
			}(),
			Ctx:         context.WithValue(context.Background(), dbtxn.ContextKey, &dbtxn.Context{}),
			ExpectedErr: "begin-error",
		},
	}
	for _, tt := range testcases {
		t.Run(tt.TestName, func(t *testing.T) {
			handler, err := dbtxn.Use(tt.Ctx, tt.DB)
			if tt.ExpectedErr != "" {
				require.EqualError(t, err, tt.ExpectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.Expected, handler)
			}
		})
	}
}

func TestUse_success(t *testing.T) {

	ctx := context.WithValue(context.Background(), dbtxn.ContextKey, &dbtxn.Context{TxMap: make(map[*sql.DB]dbtxn.Tx)})
	db, mock, _ := sqlmock.New()

	var handler *dbtxn.UseHandler
	var err error
	t.Run("trigger begin transaction when no transaction object", func(t *testing.T) {

		mock.ExpectBegin()

		handler, err = dbtxn.Use(ctx, db)

		require.NoError(t, err)
		require.Equal(t, map[*sql.DB]dbtxn.Tx{
			db: handler.DB.(dbtxn.Tx),
		}, handler.Context.TxMap)
	})

	t.Run("using available transaction", func(t *testing.T) {
		handler2, err := dbtxn.Use(ctx, db)

		require.NoError(t, err)
		require.Equal(t, handler, handler2)
	})

}

func TestAppendError(t *testing.T) {
	ctx := context.Background()
	t.Run("no txn error before begin", func(t *testing.T) {
		require.Nil(t, dbtxn.Error(ctx))
	})

	t.Run("append multiple error", func(t *testing.T) {
		dbtxn.Begin(&ctx)

		db, mock, _ := sqlmock.New()
		mock.ExpectBegin()
		handler, err := dbtxn.Use(ctx, db)
		require.NoError(t, err)

		require.True(t, handler.AppendError(errors.New("some-error-1")))
		require.False(t, handler.AppendError(nil))
		require.True(t, handler.AppendError(errors.New("some-error-2")))
		require.EqualError(t, dbtxn.Error(ctx), "some-error-1; some-error-2")
	})
}
