// A generated module for Basics functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/basics/internal/dagger"
	"fmt"
)

type Basics struct{}

// Returns a container that echoes whatever string argument is provided
func (m *Basics) ContainerEcho(stringArg string) *dagger.Container {
	return dag.Container().From("alpine:latest").WithExec([]string{"echo", stringArg})
}

// Returns lines that match a pattern in the files of the provided Directory
func (m *Basics) GrepDir(ctx context.Context, directoryArg *dagger.Directory, pattern string) (string, error) {
	return dag.Container().
		From("alpine:latest").
		WithMountedDirectory("/mnt", directoryArg).
		WithWorkdir("/mnt").
		WithExec([]string{"grep", "-R", pattern, "."}).
		Stdout(ctx)
}

func (m *Basics) RunAllChecks(ctx context.Context, source *dagger.Directory) error {
	// 1. 定義基礎環境 (鎖定 Go 1.24)
	goBase := dag.Container().
		From("golang:1.24-bookworm").
		WithDirectory("/src", source).
		WithWorkdir("/src")

	// 2. 執行 go mod tidy 檢查
	// 如果 tidy 後有變動，這步會失敗，達到 check-mod-tidy 的效果
	_, err := goBase.
		WithExec([]string{"go", "mod", "tidy"}).
		WithExec([]string{"git", "diff", "--exit-code", "go.mod", "go.sum"}).
		Sync(ctx)
	if err != nil {
		return err
	}

	// 3. 執行 golangci-lint (包含你設定的 5m timeout)
	_, err = dag.Container().
		From("golangci/golangci-lint:v2.8-alpine").
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"golangci-lint", "run", "--timeout", "5m"}).
		Sync(ctx)
	if err != nil {
		return err
	}

	// 4. 執行 govulncheck
	_, err = goBase.
		WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"}).
		WithExec([]string{"govulncheck", "./..."}).
		Sync(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (m *Basics) Ci(ctx context.Context, source *dagger.Directory) error {

	mongoSvc := dag.Container().
		From("mongo:6.0").
		WithExposedPort(27017).
		AsService()

	// 1. 建立基礎環境 (使用 Go 1.24)
	// 原本 YAML 提取 go.mod 版本的邏輯，在這裡直接鎖定環境更穩定
	goBase := dag.Container().
		From("golang:1.24-bookworm").
		WithDirectory("/src", source).
		WithWorkdir("/src")

	// 2. 執行 Lint
	// 替代原本的 golangci-lint-action
	fmt.Println("🚀 Running Lint...")
	_, err := dag.Container().
		From("golangci/golangci-lint:v2.8-alpine").
		WithDirectory("/src", source).
		WithWorkdir("/src").
		WithExec([]string{"golangci-lint", "run", "--timeout", "5m"}).
		Sync(ctx)
	if err != nil {
		return fmt.Errorf("lint failed: %w", err)
	}

	// 3. 執行測試 (比照你原本的 go test 參數)
	// -race, -count=1, -failfast, -coverprofile
	fmt.Println("🧪 Running Tests...")
	unitTestContainer := goBase.
		WithServiceBinding("mongodb", mongoSvc).
		WithEnvVariable("TEST_MONGO_URI", "mongodb://mongodb:27017").
		WithExec([]string{"go", "test", "-race", "-count=1", "-failfast", "-coverprofile=coverage.out", "./..."})

	_, err = unitTestContainer.Sync(ctx)
	if err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	// 4. 檢查覆蓋率
	// 這裡呼叫 vladopajic/go-test-coverage 的工具
	fmt.Println("📊 Checking Coverage...")
	_, err = unitTestContainer.
		WithExec([]string{"go", "install", "github.com/vladopajic/go-test-coverage/v2@latest"}).
		WithExec([]string{"go-test-coverage", "--config", "./.testcoverage.yaml", "--profile", "coverage.out"}).
		Sync(ctx)
	if err != nil {
		return fmt.Errorf("coverage check failed: %w", err)
	}

	return nil
}
