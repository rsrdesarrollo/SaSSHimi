.PHONY: bump-patch bump-minor bump-major help

# Default target
help:
	@echo "Available targets:"
	@echo "  bump-patch  - Increment patch version (x.y.Z)"
	@echo "  bump-minor  - Increment minor version (x.Y.0)"
	@echo "  bump-major  - Increment major version (X.0.0)"
	@echo ""
	@echo "Example: make bump-patch"

# Get current version from version/info.go
CURRENT_VERSION := $(shell grep 'VersionTag = ' version/info.go | cut -d'"' -f2 | sed 's/v//')

# Extract version components
MAJOR := $(shell echo $(CURRENT_VERSION) | cut -d'.' -f1)
MINOR := $(shell echo $(CURRENT_VERSION) | cut -d'.' -f2)
PATCH := $(shell echo $(CURRENT_VERSION) | cut -d'.' -f3)

# Calculate new versions
NEW_PATCH := $(shell echo $$(($(PATCH) + 1)))
NEW_MINOR := $(shell echo $$(($(MINOR) + 1)))
NEW_MAJOR := $(shell echo $$(($(MAJOR) + 1)))

# Version bump targets
bump-patch:
	@echo "Current version: v$(CURRENT_VERSION)"
	@echo "Bumping patch version to: v$(MAJOR).$(MINOR).$(NEW_PATCH)"
	@sed -i.bak 's/var VersionTag = "v$(CURRENT_VERSION)"/var VersionTag = "v$(MAJOR).$(MINOR).$(NEW_PATCH)"/' version/info.go
	@rm -f version/info.go.bak
	@git add .
	@git commit -m "Bump version to v$(MAJOR).$(MINOR).$(NEW_PATCH)"
	@git tag v$(MAJOR).$(MINOR).$(NEW_PATCH)
	@git push origin master
	@git push origin v$(MAJOR).$(MINOR).$(NEW_PATCH)
	@echo "Successfully bumped and pushed version v$(MAJOR).$(MINOR).$(NEW_PATCH)"

bump-minor:
	@echo "Current version: v$(CURRENT_VERSION)"
	@echo "Bumping minor version to: v$(MAJOR).$(NEW_MINOR).0"
	@sed -i.bak 's/var VersionTag = "v$(CURRENT_VERSION)"/var VersionTag = "v$(MAJOR).$(NEW_MINOR).0"/' version/info.go
	@rm -f version/info.go.bak
	@git add .
	@git commit -m "Bump version to v$(MAJOR).$(NEW_MINOR).0"
	@git tag v$(MAJOR).$(NEW_MINOR).0
	@git push origin master
	@git push origin v$(MAJOR).$(NEW_MINOR).0
	@echo "Successfully bumped and pushed version v$(MAJOR).$(NEW_MINOR).0"

bump-major:
	@echo "Current version: v$(CURRENT_VERSION)"
	@echo "Bumping major version to: v$(NEW_MAJOR).0.0"
	@sed -i.bak 's/var VersionTag = "v$(CURRENT_VERSION)"/var VersionTag = "v$(NEW_MAJOR).0.0"/' version/info.go
	@rm -f version/info.go.bak
	@git add .
	@git commit -m "Bump version to v$(NEW_MAJOR).0.0"
	@git tag v$(NEW_MAJOR).0.0
	@git push origin master
	@git push origin v$(NEW_MAJOR).0.0
	@echo "Successfully bumped and pushed version v$(NEW_MAJOR).0.0"
