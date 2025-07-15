package todo

import (
	"time"
)

// Tag represents a tag that can be applied to todos
type Tag struct {
	_ struct{} `dbdef:"table:tags;unique:uk_tags_name,name"`

	ID        string    `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name      string    `db:"name" dbdef:"type:varchar(50);not_null;unique"`
	Color     string    `db:"color" dbdef:"type:varchar(7);not_null;default:'#0066cc'"`
	CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`

	// Relationships
	Todos []Todo `db:"-" orm:"has_many_through:Todo,join_table:todo_tags,source_fk:tag_id,target_fk:todo_id"`
	Users []User `db:"-" orm:"has_many_through:User,join_table:user_tags,source_fk:tag_id,target_fk:user_id"`
}

// TodoTag represents the many-to-many relationship between todos and tags
type TodoTag struct {
	_ struct{} `dbdef:"table:todo_tags"`

	TodoID    string    `db:"todo_id" dbdef:"type:uuid;not_null;primary_key;foreign_key:todos.id;on_delete:CASCADE"`
	TagID     string    `db:"tag_id" dbdef:"type:uuid;not_null;primary_key;foreign_key:tags.id;on_delete:CASCADE"`
	CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
}

// UserTag represents the many-to-many relationship between users and tags
type UserTag struct {
	_ struct{} `dbdef:"table:user_tags"`

	UserID    string    `db:"user_id" dbdef:"type:uuid;not_null;primary_key;foreign_key:users.id;on_delete:CASCADE"`
	TagID     string    `db:"tag_id" dbdef:"type:uuid;not_null;primary_key;foreign_key:tags.id;on_delete:CASCADE"`
	CreatedAt time.Time `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
}
