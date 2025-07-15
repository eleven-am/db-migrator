package todo

import (
	"time"
)

// TodoStatus represents the status of a todo item
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
	TodoStatusCancelled  TodoStatus = "cancelled"
)

// TodoPriority represents the priority level of a todo
type TodoPriority string

const (
	TodoPriorityLow    TodoPriority = "low"
	TodoPriorityMedium TodoPriority = "medium"
	TodoPriorityHigh   TodoPriority = "high"
	TodoPriorityUrgent TodoPriority = "urgent"
)

// Todo represents a todo item
type Todo struct {
	_ struct{} `dbdef:"table:todos;index:idx_todos_user,user_id;index:idx_todos_category,category_id;index:idx_todos_status,status;index:idx_todos_due_date,due_date"`

	ID          string       `db:"id" dbdef:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      string       `db:"user_id" dbdef:"type:uuid;not_null;foreign_key:users.id;on_delete:CASCADE"`
	CategoryID  *string      `db:"category_id" dbdef:"type:uuid;foreign_key:categories.id;on_delete:SET NULL"`
	Title       string       `db:"title" dbdef:"type:varchar(255);not_null"`
	Description *string      `db:"description" dbdef:"type:text"`
	Status      TodoStatus   `db:"status" dbdef:"type:varchar(20);not_null;default:'pending'"`
	Priority    TodoPriority `db:"priority" dbdef:"type:varchar(20);not_null;default:'medium'"`
	DueDate     *time.Time   `db:"due_date" dbdef:"type:timestamptz"`
	CompletedAt *time.Time   `db:"completed_at" dbdef:"type:timestamptz"`
	CreatedAt   time.Time    `db:"created_at" dbdef:"type:timestamptz;not_null;default:now()"`
	UpdatedAt   time.Time    `db:"updated_at" dbdef:"type:timestamptz;not_null;default:now()"`

	// Relationships
	User     *User     `db:"-" orm:"belongs_to:User,foreign_key:user_id"`
	Category *Category `db:"-" orm:"belongs_to:Category,foreign_key:category_id"`
	Tags     []Tag     `db:"-" orm:"has_many_through:Tag,join_table:todo_tags,source_fk:todo_id,target_fk:tag_id"`
	Comments []Comment `db:"-" orm:"has_many:Comment,foreign_key:todo_id"`
}
