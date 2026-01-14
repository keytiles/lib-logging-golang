package kt_logging

import "go.uber.org/zap"

// constructs a String label with the given key and value.
func StringLabel(key string, val string) Label {
	return Label{key: key, _type: StringType, stringValue: val}
}

// constructs a Float label with the given key and value.
func FloatLabel(key string, val float64) Label {
	return Label{key: key, _type: FloatType, floatValue: val}
}

// constructs an Integer label with the given key and value.
func IntLabel(key string, val int64) Label {
	return Label{key: key, _type: IntType, intValue: val}
}

// constructs a Boolean label with the given key and value.
func BoolLabel(key string, val bool) Label {
	return Label{key: key, _type: BoolType, boolValue: val}
}

// A LabelType indicates the data type the Label carries
type LabelType uint8

const (
	// UnknownType is the default Label type. Attempting to add it to an encoder will panic.
	UnknownType LabelType = iota
	// BoolType indicates that the Label carries a bool.
	BoolType
	// FloatType indicates that the Label carries a float64.
	FloatType
	// IntType indicates that the Label carries an int64.
	IntType
	// StringType indicates that the Label carries a string.
	StringType
)

// A Label can carry a certain key-value pair. Regarding the value the atomic JSON data types are supported: string, bool, numeric values.
// Complex structures like array/object are not as we do not want to bring this complexity as a key-value into central persistence layers (like Elastic Search
// or similar).
// You can marshal complex structures into a JSON string before logging and just log the string representation
type Label struct {
	key         string
	_type       LabelType
	intValue    int64
	floatValue  float64
	stringValue string
	boolValue   bool
}

func (l Label) GetKey() string {
	return l.key
}
func (l Label) GetType() LabelType {
	return l._type
}
func (l Label) GetStringValue() string {
	return l.stringValue
}
func (l Label) GetBoolValue() bool {
	return l.boolValue
}
func (l Label) GetIntValue() int64 {
	return l.intValue
}
func (l Label) GetFloatValue() float64 {
	return l.floatValue
}

// converts a Label into zap.Field struct
func (f Label) toZapField() zap.Field {
	switch f._type {
	case BoolType:
		return zap.Bool(f.key, f.boolValue)
	case IntType:
		return zap.Int64(f.key, f.intValue)
	case FloatType:
		return zap.Float64(f.key, f.floatValue)
	case StringType:
		return zap.String(f.key, f.stringValue)
	default:
		// this normally should not happen but in case does lets just make it visible in the logs something is fishy...
		return zap.String(f.key, "!unknown_type!")
	}
}

// converts a set of Labels into an equivalent set of zap.Fields
func toZapFieldArray(fieldArray []Label) []zap.Field {
	result := []zap.Field{}
	for _, field := range fieldArray {
		result = append(result, field.toZapField())
	}
	return result
}
