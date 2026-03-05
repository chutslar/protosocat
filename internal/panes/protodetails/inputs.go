package protodetails

import (
	"fmt"
	"log"
	"protosocat/internal/colors"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func GetEditor(field protoreflect.FieldDescriptor) FieldEditor {
	switch field.Kind() {
	case protoreflect.StringKind, protoreflect.BytesKind:
		ta := textarea.New()
		ta.Prompt = ""
		ta.ShowLineNumbers = false
		return &TextArea{
			ta: ta,
		}
	case protoreflect.BoolKind:
		return Checkmark{}
	case protoreflect.DoubleKind,
		protoreflect.Fixed32Kind,
		protoreflect.Fixed64Kind,
		protoreflect.FloatKind,
		protoreflect.Int32Kind,
		protoreflect.Int64Kind,
		protoreflect.Sfixed32Kind,
		protoreflect.Sfixed64Kind,
		protoreflect.Sint32Kind,
		protoreflect.Sint64Kind,
		protoreflect.Uint32Kind,
		protoreflect.Uint64Kind:
		ti := textinput.New()
		ti.Prompt = ""
		if field.Cardinality() == protoreflect.Optional {
			return &TextInput{
				ti:        ti,
				validator: OptionalNumericValidator,
			}
		}
		return &TextInput{
			ti:        ti,
			validator: NumericValidator,
		}
	case protoreflect.EnumKind:
		return NewEnumPicker(field.Enum())
	default:
		ti := textinput.New()
		ti.Prompt = ""
		return &TextInput{
			ti: ti,
		}
	}
}

var InputStyle lipgloss.Style = lipgloss.NewStyle().
	Width(40).
	Border(lipgloss.NormalBorder(), false, false, true, false).
	BorderForeground(colors.BorderColor)

func NumericValidator(e FieldEditor) bool {
	_, err := strconv.ParseFloat(e.ValueString(), 32)
	return err == nil
}

func OptionalNumericValidator(e FieldEditor) bool {
	if e.ValueString() == "" {
		return true
	}
	return NumericValidator(e)
}

// Interface for editors that map to a protobuf field.
type FieldEditor interface {
	Update(tea.Msg) (FieldEditor, tea.Cmd)
	View() string
	ValueString() string
	Focus()
	Blur()
	Validate() bool
	ProtoValue(protoreflect.FieldDescriptor) (*protoreflect.Value, error)
	IsEmpty() bool
}

// TextInput is used for protobuf strings and bytes fields.
type TextInput struct {
	ti        textinput.Model
	validator func(FieldEditor) bool
}

func (t *TextInput) Blur() {
	t.ti.Blur()
}

func (t *TextInput) Focus() {
	t.ti.Focus()
}

func (t *TextInput) Update(msg tea.Msg) (FieldEditor, tea.Cmd) {
	var cmd tea.Cmd
	t.ti, cmd = t.ti.Update(msg)
	return t, cmd
}

func (t *TextInput) Validate() bool {
	if t.validator != nil {
		return t.validator(t)
	}
	return true
}

func (t TextInput) View() string {
	return t.ti.View()
}

func (t TextInput) ValueString() string {
	return t.ti.Value()
}

func (t TextInput) ProtoValue(d protoreflect.FieldDescriptor) (*protoreflect.Value, error) {
	if t.ti.Value() == "" {
		if d.Cardinality() == protoreflect.Optional {
			return nil, nil
		} else {
			return nil, fmt.Errorf("field %s was required but empty", d.Name())
		}
	}
	switch d.Kind() {
	case protoreflect.DoubleKind:
		f, err := strconv.ParseFloat(t.ti.Value(), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for double %s: %s", d.Name(), t.ti.Value())
		}
		v := protoreflect.ValueOfFloat64(f)
		return &v, nil
	case protoreflect.Fixed32Kind, protoreflect.Uint32Kind:
		u, err := strconv.ParseUint(t.ti.Value(), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid value for fixed32 %s: %s", d.Name(), t.ti.Value())
		}
		v := protoreflect.ValueOfUint32(uint32(u))
		return &v, nil
	case protoreflect.Fixed64Kind, protoreflect.Uint64Kind:
		u, err := strconv.ParseUint(t.ti.Value(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for fixed64 %s: %s", d.Name(), t.ti.Value())
		}
		v := protoreflect.ValueOfUint64(u)
		return &v, nil
	case protoreflect.FloatKind:
		f, err := strconv.ParseFloat(t.ti.Value(), 32)
		if err != nil {
			return nil, fmt.Errorf("invalid value for float %s: %s", d.Name(), t.ti.Value())
		}
		v := protoreflect.ValueOfFloat32(float32(f))
		return &v, nil
	case protoreflect.Int32Kind, protoreflect.Sfixed32Kind, protoreflect.Sint32Kind:
		i, err := strconv.ParseInt(t.ti.Value(), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid value for int32 %s: %s", d.Name(), t.ti.Value())
		}
		v := protoreflect.ValueOfInt32(int32(i))
		return &v, nil
	case protoreflect.Int64Kind, protoreflect.Sfixed64Kind, protoreflect.Sint64Kind:
		i, err := strconv.ParseInt(t.ti.Value(), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for int64 %s: %s", d.Name(), t.ti.Value())
		}
		v := protoreflect.ValueOfInt64(i)
		return &v, nil
	default:
		return nil, fmt.Errorf("invalid kind for %s: %s", d.Name(), d.Kind())
	}
}

func (t TextInput) IsEmpty() bool {
	return t.ti.Value() == ""
}

func (t TextInput) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "\"TextInput='%s'\"", t.ValueString()), nil
}

// TextArea is used for protobuf numeric and enum fields.
type TextArea struct {
	ta textarea.Model
}

func (t *TextArea) Blur() {
	t.ta.Blur()
}

func (t *TextArea) Focus() {
	t.ta.Focus()
}

func (t *TextArea) Update(msg tea.Msg) (FieldEditor, tea.Cmd) {
	var cmd tea.Cmd
	t.ta, cmd = t.ta.Update(msg)
	return t, cmd
}

func (t *TextArea) Validate() bool {
	return true
}

func (t TextArea) View() string {
	return t.ta.View()
}

func (t TextArea) ValueString() string {
	return t.ta.Value()
}

func (t TextArea) ProtoValue(d protoreflect.FieldDescriptor) (*protoreflect.Value, error) {
	if t.ta.Value() == "" {
		if d.Cardinality() == protoreflect.Optional {
			return nil, nil
		} else {
			return nil, fmt.Errorf("field %s was required but empty", d.Name())
		}
	}
	switch d.Kind() {
	case protoreflect.BytesKind:
		v := protoreflect.ValueOfBytes([]byte(t.ta.Value()))
		return &v, nil
	case protoreflect.StringKind:
		v := protoreflect.ValueOfString(t.ta.Value())
		return &v, nil
	default:
		return nil, fmt.Errorf("invalid kind for protobuf field (%s, %s)", d.Kind(), d.Name())
	}
}

func (t TextArea) IsEmpty() bool {
	return t.ta.Value() == ""
}

func (t TextArea) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "\"TextArea='%s'\"", t.ValueString()), nil
}

// Checkmark is used for protobuf boolean fields.
type Checkmark struct {
	Value bool
}

func (c Checkmark) Update(msg tea.Msg) (FieldEditor, tea.Cmd) {

	log.Println("update was called")
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "space" {
			c.Value = !c.Value
		}
	}
	return c, nil
}

func (c Checkmark) View() string {
	if c.Value {
		return "[X]"
	}
	return "[ ]"
}

func (c Checkmark) Focus() {}

func (c Checkmark) Blur() {}

func (c Checkmark) Validate() bool {
	return true
}

func (c Checkmark) ValueString() string {
	return strconv.FormatBool(c.Value)
}

func (c Checkmark) ProtoValue(d protoreflect.FieldDescriptor) (*protoreflect.Value, error) {
	if d.Kind() == protoreflect.BoolKind {
		v := protoreflect.ValueOfBool(c.Value)
		return &v, nil
	}
	return nil, fmt.Errorf("invalid kind for bool field: %s", d.Kind())
}

func (c Checkmark) IsEmpty() bool {
	return false
}

func (c Checkmark) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "\"Checkmark = %s\"", c.ValueString()), nil
}

type EnumPicker struct {
	options         []string
	selectedIndex   int
	maxOptionLength int
}

func NewEnumPicker(d protoreflect.EnumDescriptor) EnumPicker {
	var options []string
	maxOptionLength := 0
	for i := range d.Values().Len() {
		option := string(d.Values().Get(i).Name())
		maxOptionLength = max(len(option), maxOptionLength)
		options = append(options, option)
	}

	return EnumPicker{
		options:         options,
		selectedIndex:   0,
		maxOptionLength: maxOptionLength,
	}
}

func (e *EnumPicker) Left() {
	if e.selectedIndex > 0 {
		e.selectedIndex--
	}
}

func (e *EnumPicker) Right() {
	if e.selectedIndex < len(e.options)-1 {
		e.selectedIndex++
	}
}

func (e EnumPicker) Blur() {

}

func (e EnumPicker) Focus() {

}

func (e EnumPicker) Update(msg tea.Msg) (FieldEditor, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "left":
			e.Left()
		case "right":
			e.Right()
		}
	}
	return e, nil
}

func (e EnumPicker) Validate() bool {
	return true
}

func (e EnumPicker) ValueString() string {
	return e.options[e.selectedIndex]
}

func (e EnumPicker) View() string {
	prefix := "  "
	if e.selectedIndex > 0 {
		prefix = "◀ "
	}
	suffix := ""
	if e.selectedIndex < len(e.options)-1 {
		suffix = " ▶"
	}

	option := e.options[e.selectedIndex]
	if len(option) < e.maxOptionLength {
		padding := e.maxOptionLength - len(option)
		leftPad := padding / 2
		rightPad := padding - leftPad
		option = strings.Repeat(" ", leftPad) + option + strings.Repeat(" ", rightPad)
	}

	return fmt.Sprintf("%s%s%s", prefix, option, suffix)
}

func (e EnumPicker) ProtoValue(d protoreflect.FieldDescriptor) (*protoreflect.Value, error) {
	if d.Kind() == protoreflect.EnumKind {
		v := protoreflect.ValueOfEnum(protoreflect.EnumNumber(e.selectedIndex))
		return &v, nil
	}
	return nil, fmt.Errorf("invalid kind for enum field: %s", d.Kind())
}

func (e EnumPicker) IsEmpty() bool {
	return false
}
