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
	config *ststorm.Config
	logger ststorm.Logger
}

func NewORM(config *ststorm.Config, logger ststorm.Logger) *ORMImpl {
	return &ORMImpl{
		config: config,
		logger: logger,
	}
}

func (o *ORMImpl) Generate(ctx context.Context, opts ststorm.GenerateOptions) error {
	o.logger.Info("Generating ORM code...", "package", opts.PackagePath)

	config := orm_generator.GenerationConfig{
		PackageName:  filepath.Base(opts.PackagePath),
		OutputDir:    opts.OutputDir,
		IncludeTests: opts.IncludeTests,
		IncludeDocs:  true,
	}

	generator := orm_generator.NewCodeGenerator(config)

	if err := generator.DiscoverModels(opts.PackagePath); err != nil {
		return fmt.Errorf("failed to discover models: %w", err)
	}

	if err := generator.ValidateModels(); err != nil {
		return fmt.Errorf("failed to validate models: %w", err)
	}

	if err := generator.GenerateAll(); err != nil {
		return fmt.Errorf("failed to generate ORM code: %w", err)
	}

	models := generator.GetModelNames()
	o.logger.Info("ORM code generated successfully", "models", len(models))
	return nil
}
