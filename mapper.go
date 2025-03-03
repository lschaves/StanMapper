package StanMapper

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RealTypesMapper struct {
	RealTypesNames []string
}

// NewRealTypes creates and initializes a RealTypes instance
func NewRealTypes() *RealTypesMapper {
	return &RealTypesMapper{
		RealTypesNames: []string{
			"string",         // String
			"uint8",          // Byte
			"int32",          // Int32
			"int64",          // Int64
			"int8",           // SByte
			"uuid.UUID",      // Guid
			"float64",        // Decimal/Double
			"int",            // Integer
			"int16",          // Int16
			"bool",           // Boolean
			"time.Time",      // Date/DateTime
			"sql.NullInt64",  // Nullable Int64 (similar to Nullable in C#)
			"sql.NullString", // Nullable String
			"*string",        // String pointer
			"*uint8",         // Byte
			"*int32",         // Int32
			"*int64",         // Int64
			"*int8",          // SByte
			"*uuid.UUID",     // Guid
			"*float64",       // Decimal/Double
			"*int",           // Integer
			"*int16",         // Int16
			"*bool",          // Boolean
			"*time.Time",     // Date/DateTime
			"CustomTime",     // CustomTime
			"*CustomTime",    // CustomTime Pointer
		},
	}
}

// IsInList checks if a given type name exists in RealTypesNames
func (rt *RealTypesMapper) IsInList(typeName string) bool {
	for _, item := range rt.RealTypesNames {
		if item == typeName {
			return true
		}
	}
	return false
}

func GenerateNewObjectWithConverter[T any](fromObject interface{}, fieldMappings map[string]string, converters map[string]func(interface{}) interface{}) T {
	var retVal T
	if converters == nil {
		converters = GetConverters()
	}
	if fieldMappings == nil {
		fieldMappings = map[string]string{}
	}

	toObject := reflect.New(reflect.TypeOf(retVal)).Elem() // Create a pointer to a non-pointer type instance
	generateNewObjectBasicInnerLists(fromObject, toObject.Addr().Interface(), fieldMappings, converters)
	return toObject.Interface().(T) // Return the populated struct
}

func GenerateNewObjectsWithConverter[T any, U any](fromObjects []U, fieldMappings map[string]string, converters map[string]func(interface{}) interface{}) []T {
	var result []T

	if converters == nil {
		converters = GetConverters()
	}
	if fieldMappings == nil {
		fieldMappings = map[string]string{}
	}

	for _, fromObject := range fromObjects {
		var newObj T
		toObject := reflect.New(reflect.TypeOf(newObj)).Elem() // Create a new instance of T

		// Apply field mappings from fromObject to toObject
		generateNewObjectBasicInnerLists(fromObject, toObject.Addr().Interface(), fieldMappings, converters)

		// Append the transformed object to the result slice
		result = append(result, toObject.Interface().(T))
	}

	return result
}

func generateNewObjectBasicInnerLists(fromObject interface{}, toObject interface{}, fieldMappings map[string]string, converters map[string]func(interface{}) interface{}) {
	if fromObject == nil || toObject == nil {
		return
	}

	fromVal := reflect.ValueOf(fromObject)
	toVal := reflect.ValueOf(toObject)

	if fromVal.Kind() != reflect.Struct || toVal.Kind() != reflect.Ptr || toVal.Elem().Kind() != reflect.Struct {
		panic("fromObject must be a struct and toObject must be a pointer to a struct")
	}

	// Copy fields directly
	for i := 0; i < fromVal.NumField(); i++ {
		fromField := fromVal.Field(i)
		fromFieldType := fromVal.Type().Field(i)
		toField := toVal.Elem().FieldByName(fromFieldType.Name)

		// fmt.Printf("Processing field: %s\n", fromFieldType.Name)

		if fromField.Kind() == reflect.Slice && toField.Kind() == reflect.Slice {
			// Handle slices
			if fromField.IsValid() && !fromField.IsNil() {
				fromElemType := fromField.Type().Elem()
				toElemType := toField.Type().Elem()

				// Check if the elements are structs for nested processing
				if fromElemType.Kind() == reflect.Struct && toElemType.Kind() == reflect.Struct {
					newSlice := reflect.MakeSlice(toField.Type(), 0, fromField.Len())
					for j := 0; j < fromField.Len(); j++ {
						fromElem := fromField.Index(j)
						toElem := reflect.New(toElemType).Elem()

						generateNewObjectBasicInnerLists(fromElem.Interface(), toElem.Addr().Interface(), fieldMappings, converters)
						newSlice = reflect.Append(newSlice, toElem)
					}
					toField.Set(newSlice)
				} else if fromElemType == toElemType {
					// Directly copy if element types match
					toField.Set(fromField)
				}
			}
		} else if fromField.Kind() == reflect.Struct && toField.Kind() == reflect.Struct {
			// Handle nested structs
			generateNewObjectBasicInnerLists(fromField.Interface(), toField.Addr().Interface(), fieldMappings, converters)
		} else if fromField.Kind() == reflect.Struct && toField.Kind() == reflect.Ptr && toField.Elem().Kind() == reflect.Struct {
			// Create a new struct for a nil pointer field
			if toField.IsNil() {
				toField.Set(reflect.New(toField.Type().Elem()))
			}
			generateNewObjectBasicInnerLists(fromField.Interface(), toField.Interface(), fieldMappings, converters)
		} else if toField.IsValid() && toField.CanSet() {
			fromType := fromField.Type().String()
			toType := toField.Type().String()

			converterKey := fromType + "->" + toType
			if converter, exists := converters[converterKey]; exists {
				convertedValue := converter(fromField.Interface())
				// fmt.Printf("Converted field: %s, value: %v -> %v\n", fromFieldType.Name, fromField.Interface(), convertedValue)
				toField.Set(reflect.ValueOf(convertedValue))
			} else if fromField.Type() == toField.Type() {
				// fmt.Printf("Directly set field: %s, value: %v\n", fromFieldType.Name, fromField.Interface())
				toField.Set(fromField)
			}
		}
	}

	// Copy fields using mappings (supports nested paths)
	for fromFieldPath, toFieldName := range fieldMappings {
		// fmt.Printf("Attempting to map: %s -> %s\n", fromFieldPath, toFieldName)

		fromField := resolveNestedField(fromVal, fromFieldPath)
		if !fromField.IsValid() {
			// fmt.Printf("Mapped field not found: %s\n", fromFieldPath)
			continue
		}

		toField := resolveNestedFieldWritable(toVal.Elem(), toFieldName)
		if !toField.IsValid() {
			// fmt.Printf("Target field invalid or cannot be set: %s\n", toFieldName)
			continue
		}

		fromType := fromField.Type().String()
		toType := toField.Type().String()

		converterKey := fromType + "->" + toType
		if converter, exists := converters[converterKey]; exists {
			convertedValue := converter(fromField.Interface())
			// fmt.Printf("Mapped and converted field: %s -> %s, value: %v -> %v\n", fromFieldPath, toFieldName, fromField.Interface(), convertedValue)
			toField.Set(reflect.ValueOf(convertedValue))
		} else if fromField.Type() == toField.Type() {
			// fmt.Printf("Mapped field: %s -> %s, value: %v\n", fromFieldPath, toFieldName, fromField.Interface())
			toField.Set(fromField)
		}
	}
}

func resolveNestedField(base reflect.Value, fieldPath string) reflect.Value {
	parts := strings.Split(fieldPath, ".")
	current := base

	for _, part := range parts {
		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}
		if current.Kind() == reflect.Slice {
			// Handle slices by taking the first element
			if current.Len() > 0 {
				current = current.Index(0) // Use the first element for mapping
			} else {
				return reflect.Value{}
			}
		}
		if current.Kind() == reflect.Struct {
			current = current.FieldByName(part)
		} else {
			return reflect.Value{}
		}
	}
	return current
}

func resolveNestedFieldWritable(base reflect.Value, fieldPath string) reflect.Value {
	parts := strings.Split(fieldPath, ".")
	current := base

	for _, part := range parts {
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				current.Set(reflect.New(current.Type().Elem()))
				// fmt.Printf("Initialized pointer: %s\n", strings.Join(parts[:i+1], "."))
			}
			current = current.Elem()
		}
		if current.Kind() == reflect.Slice {
			if current.IsNil() {
				current.Set(reflect.MakeSlice(current.Type(), 0, 1))
				// fmt.Printf("Initialized slice: %s\n", strings.Join(parts[:i+1], "."))
			}
			if current.Len() == 0 {
				// Create a single element for mapping
				elem := reflect.New(current.Type().Elem()).Elem()
				current.Set(reflect.Append(current, elem))
			}
			current = current.Index(0)
		}
		if current.Kind() == reflect.Struct {
			current = current.FieldByName(part)
			if !current.IsValid() {
				// fmt.Printf("Field not found: %s\n", part)
				return reflect.Value{}
			}
		} else {
			// fmt.Printf("Failed to resolve: %s\n", strings.Join(parts[:i+1], "."))
			return reflect.Value{}
		}
	}
	return current
}

func GetConverters() map[string]func(interface{}) interface{} {
	return map[string]func(interface{}) interface{}{
		"float64->string": func(value interface{}) interface{} {
			return fmt.Sprintf("%.2f", value.(float64)) // Format as string with 2 decimal places
		},
		"float64->int64": func(value interface{}) interface{} {
			return int64(value.(float64)) // Convert float64 to int64
		},
		"*string->string": func(value interface{}) interface{} {
			if value == nil {
				return "" // Return an empty string if the pointer is nil
			}
			return *(value.(*string)) // Dereference the pointer
		},
		"string->*string": func(value interface{}) interface{} {
			str := value.(string)
			return &str // Return a pointer to the string
		},
		"*bool->bool": func(value interface{}) interface{} {
			if value == nil {
				return false
			}
			return *(value.(*bool))
		},
		"bool->*bool": func(value interface{}) interface{} {
			if value == nil {
				return false
			}
			boolVal := value.(bool)
			return &boolVal
		},
		"uuid.UUID->string": func(value interface{}) interface{} {
			return value.(uuid.UUID).String()
		},
		"CustomTime->time.Time": func(value interface{}) interface{} {
			return value.(CustomTime).Time
		},
		"CustomTime->CustomTime": func(value interface{}) interface{} {
			return value.(CustomTime)
		},
	}
}

type CustomTime struct {
	time.Time
}
