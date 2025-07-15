package todo

import (
	"context"
	"fmt"
	"github.com/eleven-am/storm/internal/orm"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// SearchParams represents search parameters
type SearchParams struct {
	UserID               string
	Statuses             []TodoStatus
	Priorities           []TodoPriority
	CategoryIDs          []string
	DueDateFrom          *time.Time
	DueDateTo            *time.Time
	SearchText           string
	IncludeUncategorized bool
}

// AdvancedQueryExamples demonstrates complex query patterns with And, Or, Not
func AdvancedQueryExamples() {
	// Connect to database
	db, err := sqlx.Connect("postgres", "postgres://user:pass@localhost:5432/todo_db?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	storm := NewStorm(db)
	ctx := context.Background()

	// Assume we have a user
	userID := "user-123"

	fmt.Println("=== Advanced Query Examples ===")

	// Example 1: Simple AND (implicit - multiple Where calls)
	fmt.Println("1. Simple AND - High priority pending todos:")
	todos1, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(Todos.Status.Eq(string(TodoStatusPending))).
		Where(Todos.Priority.Eq(string(TodoPriorityHigh))).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos1))

	// Example 2: Explicit AND using storm.And()
	fmt.Println("2. Explicit AND - Same query using storm.And():")
	todos2, err := storm.Todos.Query().
		Where(storm.And(
			Todos.UserID.Eq(userID),
			Todos.Status.Eq(string(TodoStatusPending)),
			Todos.Priority.Eq(string(TodoPriorityHigh)),
		)).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos2))

	// Example 3: OR conditions using storm.Or()
	fmt.Println("3. OR - High or Urgent priority todos:")
	todos3, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(storm.Or(
			Todos.Priority.Eq(string(TodoPriorityHigh)),
			Todos.Priority.Eq(string(TodoPriorityUrgent)),
		)).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos3))

	// Example 4: NOT condition using storm.Not()
	fmt.Println("4. NOT - Todos that are NOT completed:")
	todos4, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(storm.Not(Todos.Status.Eq(string(TodoStatusCompleted)))).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos4))

	// Example 5: Complex nested conditions - (A AND B) OR (C AND D)
	fmt.Println("5. Complex nested - (High priority AND due today) OR (Urgent priority):")
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	todos5, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(storm.Or(
			storm.And(
				Todos.Priority.Eq(string(TodoPriorityHigh)),
				Todos.DueDate.Between(startOfDay, endOfDay),
			),
			Todos.Priority.Eq(string(TodoPriorityUrgent)),
		)).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos5))

	// Example 6: Combining NOT with OR
	fmt.Println("6. NOT with OR - Todos that are NOT (completed OR cancelled):")
	todos6, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(storm.Not(
			storm.Or(
				Todos.Status.Eq(string(TodoStatusCompleted)),
				Todos.Status.Eq(string(TodoStatusCancelled)),
			),
		)).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos6))

	// Example 7: Using method chaining on conditions
	fmt.Println("7. Method chaining - High priority OR (Medium priority AND due soon):")
	tomorrow := time.Now().Add(24 * time.Hour)
	todos7, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(
			Todos.Priority.Eq(string(TodoPriorityHigh)).Or(
				Todos.Priority.Eq(string(TodoPriorityMedium)).And(
					Todos.DueDate.Lt(tomorrow),
				),
			),
		).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos7))

	// Example 8: IN and NOT IN operations
	fmt.Println("8. IN/NOT IN - High priority todos not in completed/cancelled status:")
	todos8, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(Todos.Priority.In(
			string(TodoPriorityHigh),
			string(TodoPriorityUrgent),
		)).
		Where(Todos.Status.NotIn(
			string(TodoStatusCompleted),
			string(TodoStatusCancelled),
		)).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos8))

	// Example 9: NULL checks combined with other conditions
	fmt.Println("9. NULL checks - Todos without category OR with specific category:")
	workCategoryID := "category-work-id"
	todos9, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(storm.Or(
			Todos.CategoryID.IsNull(),
			Todos.CategoryID.Eq(workCategoryID),
		)).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos9))

	// Example 10: Complex business logic query
	fmt.Println("10. Business logic - Overdue or high-priority upcoming todos:")
	now := time.Now()
	todos10, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(storm.And(
			// Not completed
			storm.Not(Todos.Status.Eq(string(TodoStatusCompleted))),
			// Either overdue OR high priority due tomorrow
			storm.Or(
				// Overdue (due date passed and not null)
				storm.And(
					Todos.DueDate.Lt(now),
					Todos.DueDate.IsNotNull(),
				),
				// High/Urgent priority due within 24 hours
				storm.And(
					Todos.DueDate.Between(now, tomorrow),
					storm.Or(
						Todos.Priority.Eq(string(TodoPriorityHigh)),
						Todos.Priority.Eq(string(TodoPriorityUrgent)),
					),
				),
			),
		)).
		OrderBy(Todos.DueDate.Asc()).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos10))

	// Example 11: Using string operations (LIKE)
	fmt.Println("11. String operations - Todos with 'meeting' in title:")
	todos11, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(Todos.Title.Like("%meeting%")).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos11))

	// Example 12: Time-based operations
	fmt.Println("12. Time operations - Todos created this week:")
	todos12, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(Todos.CreatedAt.ThisWeek()).
		OrderBy(Todos.CreatedAt.Desc()).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos this week\n\n", len(todos12))

	// Example 13: Complex reporting query
	fmt.Println("13. Complex reporting - Active todos by status and priority:")
	lastWeek := now.Add(-7 * 24 * time.Hour)
	todos13, err := storm.Todos.Query().
		Where(storm.And(
			Todos.UserID.Eq(userID),
			storm.Or(
				// Recently completed
				storm.And(
					Todos.Status.Eq(string(TodoStatusCompleted)),
					Todos.CompletedAt.Gte(lastWeek),
				),
				// Currently in progress
				Todos.Status.Eq(string(TodoStatusInProgress)),
				// Pending but high priority
				storm.And(
					Todos.Status.Eq(string(TodoStatusPending)),
					storm.Or(
						Todos.Priority.Eq(string(TodoPriorityHigh)),
						Todos.Priority.Eq(string(TodoPriorityUrgent)),
					),
				),
			),
		)).
		OrderBy(Todos.UpdatedAt.Desc()).
		Limit(20).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Found %d todos\n\n", len(todos13))

	// Example 14: Count with complex conditions
	fmt.Println("14. Count with conditions - Active todos count:")
	count, err := storm.Todos.Query().
		Where(Todos.UserID.Eq(userID)).
		Where(storm.And(
			Todos.Status.NotIn(string(TodoStatusCompleted), string(TodoStatusCancelled)),
			storm.Or(
				Todos.DueDate.IsNull(),
				Todos.DueDate.Gte(now),
			),
		)).
		Count()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Active todos: %d\n\n", count)

	// Example 15: Dynamic query building
	fmt.Println("15. Dynamic query building:")
	query := storm.Todos.Query().Where(Todos.UserID.Eq(userID))

	// Simulate filters
	var filters struct {
		Status      *string
		Priority    *string
		HasDueDate  bool
		SearchText  string
		CategoryIDs []string
	}
	filters.Status = strPtr(string(TodoStatusPending))
	filters.SearchText = "project"
	filters.HasDueDate = true

	// Build query dynamically
	if filters.Status != nil {
		query = query.Where(Todos.Status.Eq(*filters.Status))
	}
	if filters.Priority != nil {
		query = query.Where(Todos.Priority.Eq(*filters.Priority))
	}
	if filters.HasDueDate {
		query = query.Where(Todos.DueDate.IsNotNull())
	}
	if filters.SearchText != "" {
		query = query.Where(storm.Or(
			Todos.Title.Like("%"+filters.SearchText+"%"),
			Todos.Description.Like("%"+filters.SearchText+"%"),
		))
	}
	if len(filters.CategoryIDs) > 0 {
		query = query.Where(Todos.CategoryID.In(filters.CategoryIDs...))
	}

	dynamicTodos, err := query.
		OrderBy(Todos.CreatedAt.Desc()).
		Find()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Dynamic query found %d todos\n", len(dynamicTodos))

	// Example 16: Using conditions with transactions
	fmt.Println("\n16. Complex query in transaction:")
	err = storm.WithTransaction(ctx, func(txStorm *Storm) error {
		// Find all overdue todos
		overdueTodos, err := txStorm.Todos.Query().
			Where(storm.And(
				Todos.UserID.Eq(userID),
				Todos.Status.NotEq(string(TodoStatusCompleted)),
				Todos.DueDate.Lt(now),
				Todos.DueDate.IsNotNull(),
			)).
			Find()
		if err != nil {
			return err
		}

		fmt.Printf("   Found %d overdue todos to update\n", len(overdueTodos))

		// Update them (in a real app)
		for _, todo := range overdueTodos {
			todo.Priority = TodoPriorityUrgent
			// txStorm.Todos.Update(ctx, &todo)
		}

		return nil
	})
	if err != nil {
		log.Fatal("Transaction failed:", err)
	}
}

// BuildSearchConditions demonstrates building reusable condition sets
func BuildSearchConditions(storm *Storm, params SearchParams) orm.Condition {
	var conditions []orm.Condition

	// Always filter by user
	conditions = append(conditions, Todos.UserID.Eq(params.UserID))

	// Status filters
	if len(params.Statuses) > 0 {
		statusConditions := make([]orm.Condition, len(params.Statuses))
		for i, status := range params.Statuses {
			statusConditions[i] = Todos.Status.Eq(string(status))
		}
		conditions = append(conditions, storm.Or(statusConditions...))
	}

	// Priority filters
	if len(params.Priorities) > 0 {
		priorityConditions := make([]orm.Condition, len(params.Priorities))
		for i, priority := range params.Priorities {
			priorityConditions[i] = Todos.Priority.Eq(string(priority))
		}
		conditions = append(conditions, storm.Or(priorityConditions...))
	}

	// Date range
	if params.DueDateFrom != nil && params.DueDateTo != nil {
		conditions = append(conditions, Todos.DueDate.Between(*params.DueDateFrom, *params.DueDateTo))
	}

	// Search text
	if params.SearchText != "" {
		searchPattern := "%" + params.SearchText + "%"
		conditions = append(conditions, storm.Or(
			Todos.Title.Like(searchPattern),
			Todos.Description.Like(searchPattern),
		))
	}

	// Category filter with null handling
	if params.IncludeUncategorized && len(params.CategoryIDs) > 0 {
		conditions = append(conditions, storm.Or(
			Todos.CategoryID.IsNull(),
			Todos.CategoryID.In(params.CategoryIDs...),
		))
	} else if params.IncludeUncategorized {
		conditions = append(conditions, Todos.CategoryID.IsNull())
	} else if len(params.CategoryIDs) > 0 {
		conditions = append(conditions, Todos.CategoryID.In(params.CategoryIDs...))
	}

	return storm.And(conditions...)
}
