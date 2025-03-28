// Code generated by "enumer -linecomment -yaml -text -json -type=CommandType"; DO NOT EDIT.

package backy

import (
	"encoding/json"
	"fmt"
	"strings"
)

const _CommandTypeName = "scriptscriptFileremoteScriptpackageuser"

var _CommandTypeIndex = [...]uint8{0, 0, 6, 16, 28, 35, 39}

const _CommandTypeLowerName = "scriptscriptfileremotescriptpackageuser"

func (i CommandType) String() string {
	if i < 0 || i >= CommandType(len(_CommandTypeIndex)-1) {
		return fmt.Sprintf("CommandType(%d)", i)
	}
	return _CommandTypeName[_CommandTypeIndex[i]:_CommandTypeIndex[i+1]]
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _CommandTypeNoOp() {
	var x [1]struct{}
	_ = x[DefaultCT-(0)]
	_ = x[ScriptCT-(1)]
	_ = x[ScriptFileCT-(2)]
	_ = x[RemoteScriptCT-(3)]
	_ = x[PackageCT-(4)]
	_ = x[UserCT-(5)]
}

var _CommandTypeValues = []CommandType{DefaultCT, ScriptCT, ScriptFileCT, RemoteScriptCT, PackageCT, UserCT}

var _CommandTypeNameToValueMap = map[string]CommandType{
	_CommandTypeName[0:0]:        DefaultCT,
	_CommandTypeLowerName[0:0]:   DefaultCT,
	_CommandTypeName[0:6]:        ScriptCT,
	_CommandTypeLowerName[0:6]:   ScriptCT,
	_CommandTypeName[6:16]:       ScriptFileCT,
	_CommandTypeLowerName[6:16]:  ScriptFileCT,
	_CommandTypeName[16:28]:      RemoteScriptCT,
	_CommandTypeLowerName[16:28]: RemoteScriptCT,
	_CommandTypeName[28:35]:      PackageCT,
	_CommandTypeLowerName[28:35]: PackageCT,
	_CommandTypeName[35:39]:      UserCT,
	_CommandTypeLowerName[35:39]: UserCT,
}

var _CommandTypeNames = []string{
	_CommandTypeName[0:0],
	_CommandTypeName[0:6],
	_CommandTypeName[6:16],
	_CommandTypeName[16:28],
	_CommandTypeName[28:35],
	_CommandTypeName[35:39],
}

// CommandTypeString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func CommandTypeString(s string) (CommandType, error) {
	if val, ok := _CommandTypeNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _CommandTypeNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to CommandType values", s)
}

// CommandTypeValues returns all values of the enum
func CommandTypeValues() []CommandType {
	return _CommandTypeValues
}

// CommandTypeStrings returns a slice of all String values of the enum
func CommandTypeStrings() []string {
	strs := make([]string, len(_CommandTypeNames))
	copy(strs, _CommandTypeNames)
	return strs
}

// IsACommandType returns "true" if the value is listed in the enum definition. "false" otherwise
func (i CommandType) IsACommandType() bool {
	for _, v := range _CommandTypeValues {
		if i == v {
			return true
		}
	}
	return false
}

// MarshalJSON implements the json.Marshaler interface for CommandType
func (i CommandType) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface for CommandType
func (i *CommandType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("CommandType should be a string, got %s", data)
	}

	var err error
	*i, err = CommandTypeString(s)
	return err
}

// MarshalText implements the encoding.TextMarshaler interface for CommandType
func (i CommandType) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for CommandType
func (i *CommandType) UnmarshalText(text []byte) error {
	var err error
	*i, err = CommandTypeString(string(text))
	return err
}

// MarshalYAML implements a YAML Marshaler for CommandType
func (i CommandType) MarshalYAML() (interface{}, error) {
	return i.String(), nil
}

// UnmarshalYAML implements a YAML Unmarshaler for CommandType
func (i *CommandType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	var err error
	*i, err = CommandTypeString(s)
	return err
}
