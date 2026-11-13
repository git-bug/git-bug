package config

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/git-bug/git-bug/entities/common"
	"github.com/git-bug/git-bug/util/text"
)

// The labels section: how labels are presented, and which may be used. Both are
// keyed by label name, as bugs hold labels by name.

const (
	maxLabelNameLength        = 64
	maxLabelDescriptionLength = 256
)

// LabelsSection holds the definitions of the labels, and the labels allowed. A label
// doesn't need a definition to be used, and a definition doesn't allow a label.
type LabelsSection struct {
	definitions map_[common.Label, LabelsDefinition]
	allowed     set[common.Label]
}

// Get returns the definition of a label, false if it has none.
func (s LabelsSection) Get(name common.Label) (LabelsDefinition, bool) {
	definition, ok := s.definitions[name]
	return definition, ok
}

// Color returns the color of a label: the one its definition sets, or else the
// one derived from its name.
func (s LabelsSection) Color(name common.Label) common.LabelColor {
	if value, ok := s.definitions[name].Color.Get(); ok {
		if color, err := parseLabelColor(value); err == nil {
			return color
		}
	}
	return name.Color()
}

// Names returns the labels with a definition, sorted.
func (s LabelsSection) Names() []common.Label {
	return s.definitions.keys()
}

// Allowed tells whether a label may be used: any label if no label is allowed,
// otherwise only the allowed ones.
func (s LabelsSection) Allowed(name common.Label) bool {
	return len(s.allowed) == 0 || s.allowed.has(name)
}

// AllowedNames returns the allowed labels, sorted. It is empty if any label may be
// used.
func (s LabelsSection) AllowedNames() []common.Label {
	return s.allowed.keys()
}

// LabelsDefinition is how a label is presented. In a LabelsSetOperation, it holds
// the changes to the definition.
type LabelsDefinition struct {
	// Color of the label, "#rrggbb" in lowercase hex digits. Unset, the label has
	// the color derived from its name.
	Color Field[string] `json:"color,omitzero"`
	// Description of the label, non-empty, single line, of at most
	// maxLabelDescriptionLength characters. Unset, the label has no description.
	Description Field[string] `json:"description,omitzero"`
}

func (d LabelsDefinition) patch(change LabelsDefinition) LabelsDefinition {
	d.Color.patch(change.Color)
	d.Description.patch(change.Description)
	return d
}

func (d LabelsDefinition) Validate() error {
	if err := d.Color.validate(validateLabelColor); err != nil {
		return fmt.Errorf("color: %w", err)
	}
	if err := d.Description.validate(validateLabelDescription); err != nil {
		return fmt.Errorf("description: %w", err)
	}
	return nil
}

// parseLabelColor reads a color written "#rrggbb", in lowercase hex digits.
func parseLabelColor(value string) (common.LabelColor, error) {
	if len(value) != 7 || value[0] != '#' || strings.ToLower(value) != value {
		return common.LabelColor{}, fmt.Errorf("%q is not #rrggbb", value)
	}
	rgb, err := hex.DecodeString(value[1:])
	if err != nil {
		return common.LabelColor{}, fmt.Errorf("%q is not #rrggbb", value)
	}
	return common.LabelColor{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}, nil
}

func validateLabelColor(value string) error {
	_, err := parseLabelColor(value)
	return err
}

func validateLabelDescription(description string) error {
	if text.Empty(description) {
		return fmt.Errorf("empty")
	}
	if !text.SafeOneLine(description) {
		return fmt.Errorf("unsafe characters")
	}
	return validateLength(description, maxLabelDescriptionLength)
}

// validateLabelName checks a label name: valid for a bug, and of at most
// maxLabelNameLength characters. A bug may have a longer label, which can then be
// neither defined nor allowed.
func validateLabelName(name common.Label) error {
	if err := name.Validate(); err != nil {
		return fmt.Errorf("label: %w", err)
	}
	if err := validateLength(string(name), maxLabelNameLength); err != nil {
		return fmt.Errorf("label: %w", err)
	}
	return nil
}
