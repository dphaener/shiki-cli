package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FeatureSetupConfig contains configuration for setting up a new feature
type FeatureSetupConfig struct {
	FriendlyName string
	Description  string
}

// FeatureSetupResult contains the result of feature setup
type FeatureSetupResult struct {
	FeatureNumber int
	Slug          string
	FeatureDir    string
	SpecFile      string
	MetaFile      string
	ChecklistDir  string
}

// FeatureMeta represents the metadata stored in meta.json
type FeatureMeta struct {
	FeatureNumber int       `json:"feature_number"`
	Slug          string    `json:"slug"`
	FriendlyName  string    `json:"friendly_name"`
	Description   string    `json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Status        string    `json:"status"`
}

// SetupFeature creates the feature directory structure for a new specification
// Uses XDG storage via GetSpecsDir()
func SetupFeature(cfg FeatureSetupConfig) (*FeatureSetupResult, error) {
	// Get next feature number
	featureNumber, err := GetNextFeatureNumber()
	if err != nil {
		return nil, fmt.Errorf("get feature number: %w", err)
	}

	return SetupFeatureWithNumber(cfg, featureNumber)
}

// SetupFeatureWithNumber creates the feature structure using a specific feature number
func SetupFeatureWithNumber(cfg FeatureSetupConfig, featureNumber int) (*FeatureSetupResult, error) {
	// Generate slug from friendly name
	slug := CreateSlug(cfg.FriendlyName, featureNumber)

	// Use XDG storage path
	featureDir := GetSpecDir(slug)

	// Create directories
	if err := os.MkdirAll(featureDir, 0o755); err != nil {
		return nil, fmt.Errorf("create feature directory: %w", err)
	}

	checklistDir := filepath.Join(featureDir, "checklists")
	if err := os.MkdirAll(checklistDir, 0o755); err != nil {
		return nil, fmt.Errorf("create checklists directory: %w", err)
	}

	// Create spec file with initial content
	specFile := filepath.Join(featureDir, "spec.md")
	initialSpec := fmt.Sprintf(`# Feature Specification: %s

**Feature Number**: %03d
**Slug**: %s
**Created**: %s
**Status**: Draft

## Overview

[Brief description of the feature and why it matters to users/business]

## User Scenarios & Testing

[User stories and acceptance criteria]

## Requirements

### Functional Requirements

- **FR-001**: [requirement]

## Success Criteria

- **SC-001**: [measurable outcome]

## Assumptions

- **AS-001**: [assumption]

## Dependencies

- **DEP-001**: [dependency]

## Out of Scope

- **OOS-001**: [out of scope item]

## Constraints

- **CON-001**: [constraint]

## Security & Privacy

- **SEC-001**: [security consideration]

## Open Questions

- [question]
`, cfg.FriendlyName, featureNumber, slug, time.Now().Format("2006-01-02"))

	if err := AtomicWriteString(specFile, initialSpec, 0o644); err != nil {
		return nil, fmt.Errorf("write spec file: %w", err)
	}

	// Create meta.json
	metaFile := filepath.Join(featureDir, "meta.json")
	now := time.Now()
	meta := FeatureMeta{
		FeatureNumber: featureNumber,
		Slug:          slug,
		FriendlyName:  cfg.FriendlyName,
		Description:   cfg.Description,
		CreatedAt:     now,
		UpdatedAt:     now,
		Status:        "draft",
	}

	metaData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal meta.json: %w", err)
	}

	if err := AtomicWrite(metaFile, metaData, 0o644); err != nil {
		return nil, fmt.Errorf("write meta.json: %w", err)
	}

	return &FeatureSetupResult{
		FeatureNumber: featureNumber,
		Slug:          slug,
		FeatureDir:    featureDir,
		SpecFile:      specFile,
		MetaFile:      metaFile,
		ChecklistDir:  checklistDir,
	}, nil
}

// FeatureExists checks if a feature with the given slug already exists
func FeatureExists(slug string) bool {
	featureDir := GetSpecDir(slug)
	_, err := os.Stat(featureDir)
	return err == nil
}
