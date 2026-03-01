package protodetails

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"protosocat/internal/colors"
	"protosocat/internal/panes"
	"protosocat/internal/protos"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type DebugFieldDescriptor struct {
	fd *protoreflect.FieldDescriptor
}

func (d DebugFieldDescriptor) MarshalJSON() ([]byte, error) {
	if d.fd == nil {
		return []byte(`"nil"`), nil
	}
	return []byte(fmt.Sprintf("\"%p\"", d.fd)), nil
}

type FieldInput struct {
	Input         FieldEditor          `json:"input"`
	Descriptor    DebugFieldDescriptor `json:"descriptor"`
	parent        *FieldInput
	IndexInParent int           `json:"indexInParent"`
	SubFields     []*FieldInput `json:"subfields"`
}

type ProtoDetailsPane struct {
	message            *protos.Message
	rootField          *FieldInput
	active             *FieldInput
	style              lipgloss.Style
	createdMessage     *dynamicpb.Message
	showCreatedMessage bool
	messageError       error
}

func NewProtoDetailsPane() ProtoDetailsPane {
	return ProtoDetailsPane{
		style: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.BorderColor).
			Padding(1).
			Margin(1),
	}
}

func GetInputForMessage(message protoreflect.MessageDescriptor) *FieldInput {
	fieldInput := &FieldInput{}

	var inputs []*FieldInput
	for i := range message.Fields().Len() {
		inputs = append(inputs, GetInputForField(message.Fields().Get(i), fieldInput, i))
	}
	fieldInput.SubFields = inputs

	return fieldInput
}

func GetInputForField(field protoreflect.FieldDescriptor, parent *FieldInput, indexInParent int) *FieldInput {
	if field.Kind() == protoreflect.MessageKind {
		input := GetInputForMessage(field.Message())
		input.Descriptor = DebugFieldDescriptor{
			fd: &field,
		}
		input.parent = parent
		input.IndexInParent = indexInParent
		return input
	} else {
		var input FieldEditor
		switch field.Kind() {
		case protoreflect.StringKind, protoreflect.BytesKind:
			ta := textarea.New()
			ta.Prompt = ""
			ta.ShowLineNumbers = false
			input = &TextArea{
				ta: ta,
			}
		case protoreflect.BoolKind:
			input = &Checkmark{}
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
			switch field.Cardinality() {
			case protoreflect.Optional:
				input = &TextInput{
					ti:        ti,
					validator: OptionalNumericValidator,
				}
			case protoreflect.Repeated:
			case protoreflect.Required:
				input = &TextInput{
					ti:        ti,
					validator: NumericValidator,
				}
			}
		default:
			ti := textinput.New()
			ti.Prompt = ""
			input = &TextInput{
				ti: ti,
			}
		}

		return &FieldInput{
			Input: input,
			Descriptor: DebugFieldDescriptor{
				fd: &field,
			},
			parent:        parent,
			IndexInParent: indexInParent,
		}
	}
}

func GetFirstInput(root *FieldInput) *FieldInput {
	if len(root.SubFields) > 0 {
		return GetFirstInput(root.SubFields[0])
	}
	return root
}

func GetLastInput(root *FieldInput) *FieldInput {
	if len(root.SubFields) > 0 {
		return GetLastInput(root.SubFields[len(root.SubFields)-1])
	}
	return root
}

func (pd *ProtoDetailsPane) SetMessage(message *protos.Message) {
	pd.message = message
	if message == nil {
		pd.rootField = nil
		pd.active = nil
	} else {
		inputs := GetInputForMessage(message.Descriptor)
		pd.rootField = inputs
		pd.active = GetFirstInput(pd.rootField)
		pd.active.Input.Focus()

		jsonOutput, err := json.MarshalIndent(inputs, "", "  ")
		if err != nil {
			log.Printf("Couldn't serialize inputs: %v\n", err)
		} else {
			log.Println(string(jsonOutput))
		}
	}
}

func (pd *ProtoDetailsPane) UpdateSize(width int, height int) {
	pd.style = pd.style.Width(width - 2).Height(height - 2)
}

func (pd ProtoDetailsPane) Init() tea.Cmd {
	return nil
}

func (pd *ProtoDetailsPane) Up() {
	if pd.active != nil && pd.active.parent != nil {
		parent := pd.active.parent
		if pd.active.IndexInParent == 0 {
			ancestor := parent
			for ancestor != nil && ancestor.IndexInParent == 0 {
				ancestor = ancestor.parent
			}
			if ancestor != nil && ancestor.IndexInParent > 0 {
				ancestor_parent := ancestor.parent
				if ancestor_parent != nil {
					older_uncle := ancestor_parent.SubFields[ancestor.IndexInParent-1]
					pd.active.Input.Blur()
					pd.active = GetLastInput(older_uncle)
					pd.active.Input.Focus()
				}
			}
		} else {
			pd.active.Input.Blur()
			pd.active = GetLastInput(parent.SubFields[pd.active.IndexInParent-1])
			pd.active.Input.Focus()
		}
	}
}

func (pd *ProtoDetailsPane) Down() {
	if pd.active != nil && pd.active.parent != nil {
		parent := pd.active.parent
		if pd.active.IndexInParent == len(parent.SubFields)-1 {
			ancestor := parent
			for ancestor != nil &&
				ancestor.parent != nil &&
				ancestor.IndexInParent == len(ancestor.parent.SubFields)-1 {
				ancestor = ancestor.parent
			}
			if ancestor != nil && ancestor.parent != nil {
				grandparent := ancestor.parent
				if ancestor.IndexInParent < len(grandparent.SubFields)-1 {
					younger_uncle := grandparent.SubFields[ancestor.IndexInParent+1]
					pd.active.Input.Blur()
					pd.active = GetFirstInput(younger_uncle)
					pd.active.Input.Focus()
				}
			}
		} else {
			pd.active.Input.Blur()
			pd.active = GetFirstInput(parent.SubFields[pd.active.IndexInParent+1])
			pd.active.Input.Focus()
		}
	}
}

func CreateNewMessageFromInput(md protoreflect.MessageDescriptor, field *FieldInput) (*dynamicpb.Message, error) {
	if field == nil {
		return nil, errors.New("invalid nil field input")
	}
	msg := dynamicpb.NewMessage(md)
	for _, subField := range field.SubFields {
		descriptor := *subField.Descriptor.fd
		if descriptor.Kind() == protoreflect.MessageKind {
			childMessage, err := CreateNewMessageFromInput(descriptor.Message(), subField)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", descriptor.Name(), err)
			}
			msg.Set(descriptor, protoreflect.ValueOfMessage(childMessage))
		} else if descriptor.Cardinality() == protoreflect.Repeated {
			input := subField.Input
			r, ok := input.(*RepeatedEditor)
			if ok {
				list := msg.Mutable(descriptor).List()
				err := r.AppendTo(list, descriptor)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("field is repeated but input is not: %s", descriptor.Name())
			}
		} else {
			v, err := subField.Input.ProtoValue(descriptor)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", descriptor.Name(), err)
			}
			if v != nil {
				msg.Set(descriptor, *v)
			}
		}
	}
	return msg, nil
}

func (pd *ProtoDetailsPane) CreateNewMessageFromInputs() error {
	msg, err := CreateNewMessageFromInput(pd.message.Descriptor, pd.rootField)
	if err != nil {
		return err
	}
	pd.createdMessage = msg
	return nil
}

func (pd ProtoDetailsPane) Update(msg tea.Msg) (ProtoDetailsPane, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+b":
			if pd.showCreatedMessage {
				pd.showCreatedMessage = false
				return pd, nil
			} else {
				return pd, panes.SwitchToList()
			}
		case "up":
			pd.Up()
			return pd, nil
		case "down":
			pd.Down()
			return pd, nil
		case "ctrl+s":
			if pd.showCreatedMessage {
				// TODO send message
			} else {
				err := pd.CreateNewMessageFromInputs()
				if err != nil {
					pd.messageError = err
				} else {
					pd.messageError = nil
				}
				pd.showCreatedMessage = true
				return pd, nil
			}
		}
	}

	if pd.active != nil {
		var cmd tea.Cmd
		pd.active.Input, cmd = pd.active.Input.Update(msg)
		return pd, cmd
	}

	return pd, nil
}

func (pd ProtoDetailsPane) ViewField(field *FieldInput) string {
	var s string
	if field.Descriptor.fd == nil {
		// This is only the case for the parent
		var strs []string
		for i := range field.SubFields {
			strs = append(strs, pd.ViewField(field.SubFields[i]))
		}

		s = lipgloss.JoinVertical(lipgloss.Top, strs...)
	} else {
		fieldDescriptor := *field.Descriptor.fd
		isMessage := fieldDescriptor.Kind() == protoreflect.MessageKind

		var kindStr string
		if isMessage {
			kindStr = string(fieldDescriptor.Message().FullName())
		} else {
			kindStr = fieldDescriptor.Kind().String()
		}

		if isMessage {
			var strs []string
			header := fmt.Sprintf("%s (%s %s):", fieldDescriptor.Name(), fieldDescriptor.Cardinality().String(), kindStr)
			strs = append(strs, header)
			for i := range field.SubFields {
				strs = append(strs, pd.ViewField(field.SubFields[i]))
			}

			s = lipgloss.JoinVertical(lipgloss.Top, strs...)
		} else {
			prefix := ""
			if field == pd.active {
				prefix = "* "
			}
			header := fmt.Sprintf("%s%s (%s %s): ", prefix, fieldDescriptor.Name(), fieldDescriptor.Cardinality().String(), kindStr)

			var borderColor lipgloss.ANSIColor
			if field.Input.Validate() {
				borderColor = colors.BorderColor
			} else {
				borderColor = colors.ErrorColor
			}

			inputView := lipgloss.NewStyle().
				Width(40).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(borderColor).
				Render(field.Input.View())
			s = lipgloss.JoinVertical(lipgloss.Left, header, inputView)
		}
	}

	return lipgloss.NewStyle().PaddingLeft(2).MarginTop(1).Render(s)
}

func (pd ProtoDetailsPane) View() string {
	if pd.rootField == nil {
		return pd.style.Render("Invalid protobuf")
	}

	header := lipgloss.NewStyle().Underline(true).Render(string(pd.message.Descriptor.FullName()))

	var main string
	if pd.showCreatedMessage {
		if pd.messageError != nil {
			main = pd.messageError.Error()
		} else {
			opt := protojson.MarshalOptions{
				Indent:        "  ",
				UseProtoNames: true,
			}
			output, err := opt.Marshal(pd.createdMessage)
			if err != nil {
				main = fmt.Sprintf("Error with protobuf: %v", err)
			} else {
				main = string(output)
			}
		}
	} else {
		main = pd.ViewField(pd.rootField)
	}

	return pd.style.Render(lipgloss.JoinVertical(lipgloss.Top, header, main))
}
