package storm

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/eleven-am/storm/internal/orm-generator"
	"github.com/eleven-am/storm/pkg/storm"
)

// ORMImpl implements ORM code generation
type ORMImpl struct {
	config *storm.Config
	logger storm.Logger
}

// NewORM creates a new ORM generator
func NewORM(config *storm.Config, logger storm.Logger) *ORMImpl {
	return &ORMImpl{
		config: config,
		logger: logger,
	}
}

// Generate creates ORM code from models
func (o *ORMImpl) Generate(ctx context.Context, opts storm.GenerateOptions) error {
	o.logger.Info("Generating ORM code...", "package", opts.PackagePath)

	// Use the existing working ORM generator
	config := orm_generator.GenerationConfig{
		PackageName:  filepath.Base(opts.PackagePath),
		OutputDir:    opts.OutputDir,
		IncludeTests: opts.IncludeTests,
		IncludeDocs:  true,
	}

	generator := orm_generator.NewCodeGenerator(config)

	// Discover models from package
	if err := generator.DiscoverModels(opts.PackagePath); err != nil {
		return fmt.Errorf("failed to discover models: %w", err)
	}

	// Validate models
	if err := generator.ValidateModels(); err != nil {
		return fmt.Errorf("failed to validate models: %w", err)
	}

	// Generate all code
	if err := generator.GenerateAll(); err != nil {
		return fmt.Errorf("failed to generate ORM code: %w", err)
	}

	models := generator.GetModelNames()
	o.logger.Info("ORM code generated successfully", "models", len(models))
	return nil
}
