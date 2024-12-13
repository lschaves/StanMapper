package StanMapper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type SourceStruct struct {
	Name   string
	Age    int
	Active bool
}

type TargetStruct struct {
	FullName string
	Age      int
	IsActive bool
}

func TestGenerateNewObjectWithConverter(t *testing.T) {
	source := SourceStruct{
		Name:   "John Doe",
		Age:    30,
		Active: true,
	}

	fieldMappings := map[string]string{
		"FullName": "Name",
		"Age":      "Age",
		"IsActive": "Active",
	}

	converters := map[string]func(interface{}) interface{}{
		"FullName": func(value interface{}) interface{} {
			return value.(string) + " (Mapped)"
		},
		"IsActive": func(value interface{}) interface{} {
			return value.(bool)
		},
	}

	result := GenerateNewObjectWithConverter[TargetStruct](source, fieldMappings, converters)

	expected := TargetStruct{
		FullName: "John Doe (Mapped)",
		Age:      30,
		IsActive: true,
	}

	assert.Equal(t, expected, result)
}
