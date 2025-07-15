package todo

import (
	"time"
)

// Comment represents a comment on a todo item
type Comment struct {
	_ struct{} `dbdef:"table:comments;index:idx_comments_todo,todo_id;index:idx_comments_user,user_id"`

	ID        string    `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
	TodoID    string    `db:"todo_id" dbdef:"type:uuid;not_null;foreign_key:todos.id;on_delete:CASCADE"`
	UserID    string    `db:"user_id" dbdef:"type:uuid;not_null;foreign_key:users.id;on_delete:CASCADE"`
	Content   string    `db:"content" dbdef:"type:text;not_null"`
	CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
	UpdatedAt time.Time `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`

	// Relationships
	Todo *Todo `db:"-" orm:"belongs_to:Todo,foreign_key:todo_id"`
	User *User `db:"-" orm:"belongs_to:User,foreign_key:user_id"`
}
