package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// AuditParameter represents a strongly-typed audit parameter value
type AuditParameter struct {
	// StringValue contains string data
	StringValue string
	// IntValue contains integer data
	IntValue int64
	// FloatValue contains floating-point data
	FloatValue float64
	// BoolValue contains boolean data
	BoolValue bool
	// JSONValue contains arbitrary JSON data for complex types
	JSONValue json.RawMessage
	// IsSet indicates which field is populated
	isSet string // "string", "int", "float", "bool", or "json"
}

// NewStringParameter creates a string audit parameter
func NewStringParameter(value string) AuditParameter {
	return AuditParameter{
		StringValue: value,
		isSet:       "string",
	}
}

// NewIntParameter creates an integer audit parameter
func NewIntParameter(value int64) AuditParameter {
	return AuditParameter{
		IntValue: value,
		isSet:    "int",
	}
}

// NewFloatParameter creates a float audit parameter
func NewFloatParameter(value float64) AuditParameter {
	return AuditParameter{
		FloatValue: value,
		isSet:      "float",
	}
}

// NewBoolParameter creates a boolean audit parameter
func NewBoolParameter(value bool) AuditParameter {
	return AuditParameter{
		BoolValue: value,
		isSet:     "bool",
	}
}

// NewJSONParameter creates a JSON audit parameter for complex data
func NewJSONParameter(value json.RawMessage) AuditParameter {
	return AuditParameter{
		JSONValue: value,
		isSet:     "json",
	}
}

// NewJSONParameterFromString creates a JSON audit parameter from a JSON string
func NewJSONParameterFromString(value string) (AuditParameter, error) {
	// Validate that the string is valid JSON
	var jsonVal json.RawMessage
	if err := json.Unmarshal([]byte(value), &jsonVal); err != nil {
		return AuditParameter{}, fmt.Errorf("invalid JSON string: %w", err)
	}
	return NewJSONParameter(jsonVal), nil
}

// GetString returns the string value if set
func (p AuditParameter) GetString() (string, bool) {
	if p.isSet == "string" {
		return p.StringValue, true
	}
	return "", false
}

// GetInt returns the int value if set
func (p AuditParameter) GetInt() (int64, bool) {
	if p.isSet == "int" {
		return p.IntValue, true
	}
	return 0, false
}

// GetFloat returns the float value if set
func (p AuditParameter) GetFloat() (float64, bool) {
	if p.isSet == "float" {
		return p.FloatValue, true
	}
	return 0, false
}

// GetBool returns the boolean value if set
func (p AuditParameter) GetBool() (bool, bool) {
	if p.isSet == "bool" {
		return p.BoolValue, true
	}
	return false, false
}

// GetJSON returns the JSON value if set
func (p AuditParameter) GetJSON() (json.RawMessage, bool) {
	if p.isSet == "json" {
		return p.JSONValue, true
	}
	return nil, false
}

// UnmarshalJSON implements json.Unmarshaler for proper JSON deserialization
func (p *AuditParameter) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		*p = NewStringParameter(strVal)
		return nil
	}

	// Try to unmarshal as number
	var numVal json.Number
	if err := json.Unmarshal(data, &numVal); err == nil {
		if intVal, err := numVal.Int64(); err == nil {
			*p = NewIntParameter(intVal)
			return nil
		}
		if floatVal, err := numVal.Float64(); err == nil {
			*p = NewFloatParameter(floatVal)
			return nil
		}
	}

	// Try to unmarshal as boolean
	var boolVal bool
	if err := json.Unmarshal(data, &boolVal); err == nil {
		*p = NewBoolParameter(boolVal)
		return nil
	}

	// Try to unmarshal as JSON object/array
	var jsonVal json.RawMessage
	if err := json.Unmarshal(data, &jsonVal); err == nil {
		*p = NewJSONParameter(jsonVal)
		return nil
	}

	return fmt.Errorf("unsupported JSON type for AuditParameter")
}

type Event struct {
	Timestamp    time.Time
	TenantID     string
	UserID       string
	Action       string
	Resource     string
	Parameters   map[string]AuditParameter
	Result       string
	ErrorMessage string
	Duration     time.Duration
	RequestID    string
	Metadata     map[string]string
}

type EventSink interface {
	Record(ctx context.Context, event Event) error
	Close() error
}

type NoOpSink struct{}

func NewNoOpSink() *NoOpSink {
	return &NoOpSink{}
}

func (s *NoOpSink) Record(ctx context.Context, event Event) error {
	return nil
}

func (s *NoOpSink) Close() error {
	return nil
}
