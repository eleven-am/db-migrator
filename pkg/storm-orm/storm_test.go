package orm

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

func TestNewStorm(t *testing.T) {

	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "postgres")
	storm := NewStorm(db)

	if storm == nil {
		t.Fatal("expected storm instance, got nil")
	}
	if storm.db != db {
		t.Error("storm db does not match input db")
	}
	if storm.executor != db {
		t.Error("storm executor should be db by default")
	}
	if storm.repositories == nil {
		t.Error("repositories map should be initialized")
	}
}

func TestStormWithTransaction(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "postgres")
	storm := NewStorm(db)

	t.Run("successful transaction", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit()

		executed := false
		err := storm.WithTransaction(context.Background(), func(txStorm *Storm) error {
			executed = true
			if txStorm == storm {
				t.Error("transaction storm should be different instance")
			}
			if txStorm.db != storm.db {
				t.Error("transaction storm should have same db")
			}

			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !executed {
			t.Error("transaction function was not executed")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations not met: %v", err)
		}
	})

	t.Run("failed transaction", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()

		expectedErr := errors.New("transaction error")
		err := storm.WithTransaction(context.Background(), func(txStorm *Storm) error {
			return expectedErr
		})

		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations not met: %v", err)
		}
	})

	t.Run("panic in transaction", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic to propagate")
			}
		}()

		storm.WithTransaction(context.Background(), func(txStorm *Storm) error {
			panic("test panic")
		})
	})
}

func TestStormWithTransactionOptions(t *testing.T) {

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "postgres")
	storm := NewStorm(db)

	t.Run("with custom options", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit()

		opts := &TransactionOptions{
			Isolation: sql.LevelSerializable,
			ReadOnly:  true,
		}

		err := storm.WithTransactionOptions(context.Background(), opts, func(txStorm *Storm) error {
			return nil
		})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("expectations not met: %v", err)
		}
	})
}

func TestStormLogicalOperators(t *testing.T) {

	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	cond1 := Condition{condition: squirrel.Eq{"name": "John"}}
	cond2 := Condition{condition: squirrel.Eq{"age": 25}}
	cond3 := Condition{condition: squirrel.Eq{"active": true}}

	t.Run("And", func(t *testing.T) {
		result := And(cond1, cond2, cond3)
		sql, _, err := result.ToSqlizer().ToSql()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if sql == "" {
			t.Error("expected non-empty SQL")
		}
	})

	t.Run("Or", func(t *testing.T) {
		result := Or(cond1, cond2)
		sql, _, err := result.ToSqlizer().ToSql()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sql == "" {
			t.Error("expected non-empty SQL")
		}
	})

	t.Run("Not", func(t *testing.T) {
		result := Not(cond1)
		sql, _, err := result.ToSqlizer().ToSql()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "NOT (name = ?)"
		if sql != expected {
			t.Errorf("expected SQL %q, got %q", expected, sql)
		}
	})

	t.Run("empty And", func(t *testing.T) {
		result := And()
		sql, _, err := result.ToSqlizer().ToSql()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "(1=1)"
		if sql != expected {
			t.Errorf("expected SQL %q for empty And, got %q", expected, sql)
		}
	})

	t.Run("empty Or", func(t *testing.T) {
		result := Or()
		sql, _, err := result.ToSqlizer().ToSql()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "(1=0)"
		if sql != expected {
			t.Errorf("expected SQL %q for empty Or, got %q", expected, sql)
		}
	})
}

func TestStormGetters(t *testing.T) {

	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer mockDB.Close()

	db := sqlx.NewDb(mockDB, "postgres")
	storm := NewStorm(db)

	t.Run("GetDB", func(t *testing.T) {
		result := storm.GetDB()
		if result != db {
			t.Error("GetDB should return the underlying database")
		}
	})

	t.Run("GetExecutor", func(t *testing.T) {
		result := storm.GetExecutor()
		if result != db {
			t.Error("GetExecutor should return db by default")
		}
	})
}
