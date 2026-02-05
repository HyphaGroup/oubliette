package container

import (
	"context"
	"fmt"
	"os"
	"sort"
)

// ImageManager handles container image operations using config-defined container types
type ImageManager struct {
	containers map[string]string // type name -> image name
	runtime    Runtime
}

// NewImageManager creates a new ImageManager with the given container type mappings
func NewImageManager(containers map[string]string, runtime Runtime) *ImageManager {
	return &ImageManager{
		containers: containers,
		runtime:    runtime,
	}
}

// GetImageName returns the image name for a container type
func (m *ImageManager) GetImageName(typeName string) (string, error) {
	imageName, ok := m.containers[typeName]
	if !ok {
		return "", fmt.Errorf("unknown container type: %s", typeName)
	}
	return imageName, nil
}

// ValidTypes returns all valid container type names sorted alphabetically
func (m *ImageManager) ValidTypes() []string {
	types := make([]string, 0, len(m.containers))
	for typeName := range m.containers {
		types = append(types, typeName)
	}
	sort.Strings(types)
	return types
}

// IsValidType checks if a container type is valid
func (m *ImageManager) IsValidType(typeName string) bool {
	_, ok := m.containers[typeName]
	return ok
}

// EnsureImageExists checks if the image for a container type exists,
// and pulls it if necessary. In dev mode (OUBLIETTE_DEV=1), returns an
// error if the image doesn't exist instead of pulling.
func (m *ImageManager) EnsureImageExists(ctx context.Context, typeName string) error {
	imageName, err := m.GetImageName(typeName)
	if err != nil {
		return err
	}

	exists, err := m.runtime.ImageExists(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to check image %s: %w", imageName, err)
	}

	if exists {
		return nil
	}

	// In dev mode, don't pull - require local images
	if os.Getenv("OUBLIETTE_DEV") == "1" {
		return fmt.Errorf("image %s not found locally (dev mode - run ./build.sh to build local images)", imageName)
	}

	// Pull the image
	fmt.Printf("📦 Pulling image %s...\n", imageName)
	if err := m.runtime.Pull(ctx, imageName); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}

	return nil
}

// EnsureAllImages ensures all configured container images exist
func (m *ImageManager) EnsureAllImages(ctx context.Context) error {
	for typeName := range m.containers {
		if err := m.EnsureImageExists(ctx, typeName); err != nil {
			return err
		}
	}
	return nil
}
