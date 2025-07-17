package todo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// ExampleUsage demonstrates how to use the generated Storm ORM
func ExampleUsage() {
	// Connect to database
	db, err := sqlx.Connect("postgres", "postgres://user:pass@localhost:5432/todo_db?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Create Storm instance
	storm := NewStorm(db)
	ctx := context.Background()

	// Example 1: Create a user
	user := &User{
		Email:    "john@example.com",
		Name:     "John Doe",
		Password: "hashed_password_here",
		IsActive: true,
	}

	if err := ststorm.Users.Create(ctx, user); err != nil {
		log.Fatal("Failed to create user:", err)
	}
	fmt.Printf("Created user: %s (ID: %s)\n", user.Name, user.ID)

	// Example 2: Create a category
	category := &Category{
		UserID:      user.ID,
		Name:        "Work",
		Color:       "#FF5733",
		Description: strPtr("Work-related tasks"),
	}

	if err := ststorm.Categorys.Create(ctx, category); err != nil {
		log.Fatal("Failed to create category:", err)
	}

	// Example 3: Create todos
	todos := []*Todo{
		{
			UserID:      user.ID,
			CategoryID:  &category.ID,
			Title:       "Complete project proposal",
			Description: strPtr("Write and submit the Q4 project proposal"),
			Status:      TodoStatusPending,
			Priority:    TodoPriorityHigh,
			DueDate:     timePtr(time.Now().Add(48 * time.Hour)),
		},
		{
			UserID:      user.ID,
			CategoryID:  &category.ID,
			Title:       "Review team performance",
			Description: strPtr("Quarterly team performance reviews"),
			Status:      TodoStatusPending,
			Priority:    TodoPriorityMedium,
			DueDate:     timePtr(time.Now().Add(7 * 24 * time.Hour)),
		},
	}

	for _, todo := range todos {
		if err := ststorm.Todos.Create(ctx, todo); err != nil {
			log.Fatal("Failed to create todo:", err)
		}
		fmt.Printf("Created todo: %s\n", todo.Title)
	}

	// Example 4: Create tags
	tags := []*Tag{
		{Name: "urgent", Color: "#FF0000"},
		{Name: "review", Color: "#0066CC"},
		{Name: "documentation", Color: "#00AA00"},
	}

	for _, tag := range tags {
		if err := ststorm.Tags.Create(ctx, tag); err != nil {
			log.Fatal("Failed to create tag:", err)
		}
	}

	// Example 5: Associate tags with todos (many-to-many)
	todoTag := &TodoTag{
		TodoID: todos[0].ID,
		TagID:  tags[0].ID, // urgent tag
	}
	if err := ststorm.TodoTags.Create(ctx, todoTag); err != nil {
		log.Fatal("Failed to create todo-tag association:", err)
	}

	// Example 6: Query todos with filters
	fmt.Println("\n--- Querying Todos ---")

	// Find all pending todos for a user
	pendingTodos, err := ststorm.Todos.Query().
		Where(Todos.UserID.Eq(user.ID)).
		Where(Todos.Status.Eq(string(TodoStatusPending))).
		OrderBy(Todos.Priority.Desc()).
		Find()

	if err != nil {
		log.Fatal("Failed to query todos:", err)
	}

	fmt.Printf("Found %d pending todos\n", len(pendingTodos))
	for _, todo := range pendingTodos {
		fmt.Printf("- %s (Priority: %s)\n", todo.Title, todo.Priority)
	}

	// Example 7: Find todos by category
	categoryTodos, err := ststorm.Todos.Query().
		Where(Todos.CategoryID.Eq(category.ID)).
		OrderBy(Todos.CreatedAt.Desc()).
		Find()

	if err != nil {
		log.Fatal("Failed to find todos by category:", err)
	}

	fmt.Printf("\nTodos in '%s' category: %d\n", category.Name, len(categoryTodos))

	// Example 8: Complex query with date range
	tomorrow := time.Now().Add(24 * time.Hour)
	nextWeek := time.Now().Add(7 * 24 * time.Hour)

	upcomingTodos, err := ststorm.Todos.Query().
		Where(Todos.UserID.Eq(user.ID)).
		Where(Todos.DueDate.Between(tomorrow, nextWeek)).
		Where(Todos.Status.NotEq(string(TodoStatusCompleted))).
		OrderBy(Todos.DueDate.Asc()).
		Find()

	if err != nil {
		log.Fatal("Failed to find upcoming todos:", err)
	}

	fmt.Printf("\nUpcoming todos (next week): %d\n", len(upcomingTodos))

	// Example 9: Update a todo
	if len(todos) > 0 {
		todos[0].Status = TodoStatusInProgress
		if err := ststorm.Todos.Update(ctx, todos[0]); err != nil {
			log.Fatal("Failed to update todo:", err)
		}
		fmt.Printf("\nUpdated todo '%s' to in-progress\n", todos[0].Title)
	}

	// Example 10: Add a comment to a todo
	comment := &Comment{
		TodoID:  todos[0].ID,
		UserID:  user.ID,
		Content: "Started working on the proposal draft",
	}

	if err := ststorm.Comments.Create(ctx, comment); err != nil {
		log.Fatal("Failed to create comment:", err)
	}
	fmt.Printf("Added comment to todo '%s'\n", todos[0].Title)

	// Example 11: Transaction example
	err = ststorm.WithTransaction(ctx, func(txStorm *Storm) error {
		// Create a new todo
		newTodo := &Todo{
			UserID:   user.ID,
			Title:    "Transactional todo",
			Status:   TodoStatusPending,
			Priority: TodoPriorityLow,
		}

		if err := txStstorm.Todos.Create(ctx, newTodo); err != nil {
			return err
		}

		// Add a comment in the same transaction
		newComment := &Comment{
			TodoID:  newTodo.ID,
			UserID:  user.ID,
			Content: "Created via transaction",
		}

		if err := txStstorm.Comments.Create(ctx, newComment); err != nil {
			return err // This will rollback the entire transaction
		}

		fmt.Println("\nTransaction completed successfully")
		return nil
	})

	if err != nil {
		log.Fatal("Transaction failed:", err)
	}

	// Example 12: Count and aggregation
	totalTodos, err := ststorm.Todos.Query().
		Where(Todos.UserID.Eq(user.ID)).
		Count()

	if err != nil {
		log.Fatal("Failed to count todos:", err)
	}

	fmt.Printf("\nTotal todos for user: %d\n", totalTodos)

	// Example 13: Check if a record exists
	exists, err := ststorm.Users.Query().
		Where(Users.Email.Eq("john@example.com")).
		Exists()

	if err != nil {
		log.Fatal("Failed to check user existence:", err)
	}

	fmt.Printf("User with email 'john@example.com' exists: %v\n", exists)

	// Example 14: Delete operations
	// Delete completed todos older than 30 days
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)

	deletedCount, err := ststorm.Todos.Query().
		Where(Todos.Status.Eq(string(TodoStatusCompleted))).
		Where(Todos.CompletedAt.Lt(thirtyDaysAgo)).
		Delete()

	if err != nil {
		log.Fatal("Failed to delete old todos:", err)
	}

	fmt.Printf("\nDeleted %d old completed todos\n", deletedCount)
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
