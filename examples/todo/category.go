package todo

import (
	"time"
)

// Category represents a category for organizing todos
type Category struct {
	_ struct{} `dbdef:"table:categories;index:idx_categories_user,user_id;unique:uk_categories_user_name,user_id,name"`

	ID          string    `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      string    `db:"user_id" dbdef:"type:uuid;not_null;foreign_key:users.id;on_delete:CASCADE"`
	Name        string    `db:"name" dbdef:"type:varchar(100);not_null"`
	Color       string    `db:"color" dbdef:"type:varchar(7);not_null;default:'#808080'"`
	Description *string   `db:"description" dbdef:"type:text"`
	CreatedAt   time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
	UpdatedAt   time.Time `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`

	// Relationships
	User  *User  `db:"-" orm:"belongs_to:User,foreign_key:user_id"`
	Todos []Todo `db:"-" orm:"has_many:Todo,foreign_key:category_id"`
}
