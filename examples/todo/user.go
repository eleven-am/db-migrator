package todo

import (
	"time"
)

// User represents a user in the todo application
type User struct {
	_ struct{} `dbdef:"table:users;index:idx_users_email,email;unique:uk_users_email,email"`

	ID        string    `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email     string    `db:"email" dbdef:"type:varchar(255);not_null;unique"`
	Name      string    `db:"name" dbdef:"type:varchar(100);not_null"`
	Password  string    `db:"password_hash" dbdef:"type:varchar(255);not_null"`
	IsActive  bool      `db:"is_active" dbdef:"type:boolean;not_null;default:true"`
	CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
	UpdatedAt time.Time `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`

	// Relationships
	Todos      []Todo     `db:"-" orm:"has_many:Todo,foreign_key:user_id"`
	Categories []Category `db:"-" orm:"has_many:Category,foreign_key:user_id"`
	Tags       []Tag      `db:"-" orm:"has_many_through:Tag,join_table:user_tags,source_fk:user_id,target_fk:tag_id"`
}
