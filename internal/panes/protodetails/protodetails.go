package protodetails

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"protosocat/internal/colors"
	"protosocat/internal/panes"
	"protosocat/internal/protos"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type DebugFieldDescriptor struct {
	fd protoreflect.Descriptor
}

func (d DebugFieldDescriptor) MarshalJSON() ([]byte, error) {
	if d.fd == nil {
		return []byte(`"nil"`), nil
	}
	return fmt.Appendf(nil, "\"%p\"", d.fd), nil
}

type FieldInput struct {
	Input         FieldEditor          `json:"input"`
	Descriptor    DebugFieldDescriptor `json:"descriptor"`
	parent        *FieldInput
	IndexInParent int           `json:"indexInParent"`
	SubFields     []*FieldInput `json:"subfields"`
	SelectedOneof *int
}

func (f *FieldInput) SelectOneof(index int) {
	oneof, isoneof := f.Descriptor.fd.(protoreflect.OneofDescriptor)
	if isoneof && f.SelectedOneof == nil {
		f.SelectedOneof = &index
		input := GetInputForField(oneof.Fields().Get(index), f, 0)
		f.SubFields = []*FieldInput{
			input,
		}
	}
}

func (f *FieldInput) ResetOneof() {
	oneof, isoneof := f.Descriptor.fd.(protoreflect.OneofDescriptor)
	if isoneof && f.SelectedOneof != nil {
		f.SelectedOneof = nil
		var newSubfields []*FieldInput
		for i := range oneof.Fields().Len() {
			fd := oneof.Fields().Get(i)
			input := &FieldInput{
				Input: Checkmark{},
				Descriptor: DebugFieldDescriptor{
					fd: fd,
				},
				parent:        f,
				IndexInParent: i,
			}
			newSubfields = append(newSubfields, input)
		}

		f.SubFields = newSubfields
	}
}

func (f *FieldInput) IsOneof() bool {
	_, isOneof := f.Descriptor.fd.(protoreflect.OneofDescriptor)
	return isOneof
}

func (f *FieldInput) UpdateParentOneof() {
	_, isCheckmark := f.Input.(Checkmark)
	isOneof := f.parent.IsOneof()
	if isOneof && isCheckmark {
		f.parent.SelectOneof(f.IndexInParent)
	}
}

func (f *FieldInput) CascadeOneofReset() *FieldInput {
	oneofAncestor := f.parent
	for oneofAncestor != nil && !oneofAncestor.IsOneof() {
		oneofAncestor = oneofAncestor.parent
	}
	if oneofAncestor != nil && oneofAncestor.IsOneof() {
		oneofAncestor.ResetOneof()
		return oneofAncestor
	}
	return nil
}

type ProtoDetailsPane struct {
	message            *protos.Message
	rootField          *FieldInput
	active             *FieldInput
	style              lipgloss.Style
	createdMessage     *dynamicpb.Message
	showCreatedMessage bool
	messageError       error
	viewport           viewport.Model
}

func NewProtoDetailsPane() ProtoDetailsPane {
	return ProtoDetailsPane{
		style: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.BorderColor).
			Padding(1).
			Margin(1),
		viewport: viewport.New(),
	}
}

func GetInputForMessage(message protoreflect.MessageDescriptor) *FieldInput {
	fieldInput := &FieldInput{}

	var inputs []*FieldInput

	seenOneofs := make([]bool, message.Oneofs().Len())

	for i := range message.Fields().Len() {
		field := message.Fields().Get(i)
		oneof := field.ContainingOneof()
		if oneof != nil {

			if !seenOneofs[oneof.Index()] {
				seenOneofs[oneof.Index()] = true

				dummy := 0
				input := &FieldInput{
					parent:        fieldInput,
					IndexInParent: len(inputs),
					Descriptor: DebugFieldDescriptor{
						fd: oneof,
					},
					SelectedOneof: &dummy,
				}

				input.ResetOneof()

				inputs = append(inputs, input)
			}
		} else {
			inputs = append(inputs, GetInputForField(field, fieldInput, len(inputs)))
		}
	}
	fieldInput.SubFields = inputs

	return fieldInput
}

func GetInputForField(field protoreflect.FieldDescriptor, parent *FieldInput, indexInParent int) *FieldInput {
	if field.Kind() == protoreflect.MessageKind {
		input := GetInputForMessage(field.Message())
		input.Descriptor = DebugFieldDescriptor{
			fd: field,
		}
		input.parent = parent
		input.IndexInParent = indexInParent
		return input
	} else {
		input := GetEditor(field)

		return &FieldInput{
			Input: input,
			Descriptor: DebugFieldDescriptor{
				fd: field,
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

	verticalOverhead := pd.style.GetVerticalFrameSize() + 1 // +1 for header
	horizontalOverhead := pd.style.GetHorizontalFrameSize()

	pd.viewport.SetHeight(height - 2 - verticalOverhead)
	pd.viewport.SetWidth(width - 2 - horizontalOverhead)
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
					var nextActive *FieldInput
					for i := ancestor.IndexInParent - 1; i >= 0; i-- {
						older_uncle := ancestor_parent.SubFields[i]
						nextActive = GetLastInput(older_uncle)
						if nextActive != nil && nextActive.Input != nil {
							break
						}
					}
					if nextActive != nil && nextActive.Input != nil {
						pd.active.Input.Blur()
						pd.active = nextActive
						pd.active.Input.Focus()
					}
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
					var nextActive *FieldInput
					for i := ancestor.IndexInParent + 1; i < len(grandparent.SubFields); i++ {
						younger_uncle := grandparent.SubFields[i]
						nextActive = GetFirstInput(younger_uncle)
						if nextActive != nil && nextActive.Input != nil {
							break
						}
					}
					if nextActive != nil && nextActive.Input != nil {
						pd.active.Input.Blur()
						pd.active = nextActive
						pd.active.Input.Focus()
					}
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
		switch descriptor := subField.Descriptor.fd.(type) {
		case protoreflect.OneofDescriptor:
			output, _ := json.Marshal(subField)
			log.Printf("oneof: %s", string(output))
			if subField.SelectedOneof != nil {
				input := subField.SubFields[0]
				chosenFD, ok := input.Descriptor.fd.(protoreflect.FieldDescriptor)
				if !ok {
					return nil, fmt.Errorf("%s: invalid oneof", input.Descriptor.fd.Name())
				}
				if chosenFD.Kind() == protoreflect.MessageKind {
					childMessage, err := CreateNewMessageFromInput(chosenFD.Message(), input)
					if err != nil {
						return nil, fmt.Errorf("%s: %w", chosenFD.Name(), err)
					}
					msg.Set(chosenFD, protoreflect.ValueOfMessage(childMessage))
				} else if input.Input != nil {
					v, err := input.Input.ProtoValue(chosenFD)
					if err != nil {
						return nil, fmt.Errorf("%s: %w", chosenFD.Name(), err)
					}
					msg.Set(chosenFD, *v)
				} else {
					return nil, fmt.Errorf("%s: invalid oneof value", chosenFD.Name())
				}
			} else {
				return nil, fmt.Errorf("%s: oneof value was not chosen", descriptor.Name())
			}
		case protoreflect.FieldDescriptor:
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
		case "ctrl+e":
			if pd.active != nil {
				resetAncestor := pd.active.CascadeOneofReset()
				if resetAncestor != nil {
					if pd.active.Input != nil {
						pd.active.Input.Blur()
					}
					pd.active = GetFirstInput(resetAncestor)
					pd.active.Input.Focus()
				}
			}
		case "space":
			if pd.active != nil && pd.active.parent != nil {
				_, isCheckmark := pd.active.Input.(Checkmark)
				if isCheckmark && pd.active.parent.IsOneof() {
					pd.active.parent.SelectOneof(pd.active.IndexInParent)
					pd.active.Input.Blur()
					pd.active = GetFirstInput(pd.active.parent)
					if pd.active != nil && pd.active.Input != nil {
						pd.active.Input.Focus()
					}
				}
			}
		}
	}

	if pd.active != nil && pd.active.Input != nil {
		var cmd tea.Cmd
		pd.active.Input, cmd = pd.active.Input.Update(msg)
		return pd, cmd
	}

	return pd, nil
}

type ViewFieldResult struct {
	content            string
	offsetBeforeActive int
	foundActive        bool
}

func (pd ProtoDetailsPane) ViewField(field *FieldInput) ViewFieldResult {
	var s string
	offsetBeforeActive := 0
	foundActive := false
	if field.Descriptor.fd == nil {
		// This is only the case for the parent
		var strs []string
		for i := range field.SubFields {
			res := pd.ViewField(field.SubFields[i])
			strs = append(strs, res.content)
			if !foundActive {
				offsetBeforeActive += res.offsetBeforeActive
				if res.foundActive {
					foundActive = true
				}
			}
		}

		s = lipgloss.JoinVertical(lipgloss.Top, strs...)
	} else {
		fieldDescriptor := field.Descriptor.fd

		isOneofOption := false
		_, isCheckmark := field.Input.(Checkmark)
		if isCheckmark && field.parent.IsOneof() {
			isOneofOption = true
		}

		var header string
		isMessage := false
		switch descriptor := fieldDescriptor.(type) {
		case protoreflect.OneofDescriptor:
			header = fmt.Sprintf("%s oneof:", descriptor.Name())
			isMessage = true
		case protoreflect.FieldDescriptor:
			isMessage = !isOneofOption && descriptor.Kind() == protoreflect.MessageKind
			if isMessage {
				kindStr := string(descriptor.Message().FullName())
				header = fmt.Sprintf("%s (%s %s):", descriptor.Name(), descriptor.Cardinality().String(), kindStr)
			} else {
				kindStr := descriptor.Kind().String()
				prefix := ""
				if field == pd.active {
					prefix = "* "
					foundActive = true
				}
				header = fmt.Sprintf("%s%s (%s %s): ", prefix, descriptor.Name(), descriptor.Cardinality().String(), kindStr)
			}
		}

		if isMessage {
			var strs []string
			offsetBeforeActive = lipgloss.Height(header)
			strs = append(strs, header)
			for i := range field.SubFields {
				res := pd.ViewField(field.SubFields[i])
				strs = append(strs, res.content)
				if !foundActive {
					offsetBeforeActive += res.offsetBeforeActive
					if res.foundActive {
						foundActive = true
					}
				}
			}

			s = lipgloss.JoinVertical(lipgloss.Top, strs...)
		} else {
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

		if !foundActive {
			offsetBeforeActive = lipgloss.Height(s) + 1
		}
		offsetBeforeActive += 1 // Add one for marginTop(1)
	}

	content := lipgloss.NewStyle().PaddingLeft(2).MarginTop(1).Render(s)
	return ViewFieldResult{
		content:            content,
		offsetBeforeActive: offsetBeforeActive,
		foundActive:        foundActive,
	}
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
		res := pd.ViewField(pd.rootField)

		pd.viewport.SetContent(res.content)

		contentHeight := lipgloss.Height(res.content)
		viewportHeight := pd.viewport.Height()
		halfHeight := viewportHeight / 2
		yOffset := 0
		if res.offsetBeforeActive > halfHeight {
			yOffset = res.offsetBeforeActive - halfHeight
		}
		maxOffset := max(contentHeight-viewportHeight, 0)
		if yOffset > maxOffset {
			yOffset = maxOffset
		}

		pd.viewport.SetYOffset(yOffset)
		main = pd.viewport.View()
	}

	return pd.style.Render(lipgloss.JoinVertical(lipgloss.Top, header, main))
}
