package protos

import (
	"context"
	"fmt"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Parser struct {
	compiler   protocompile.Compiler
	sources    []string
	shortPaths []string
}

func NewParser(directory string) *Parser {
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(
			&protocompile.SourceResolver{},
		),
	}
	return &Parser{
		compiler: compiler,
	}
}

func (p *Parser) AddSource(fullPath string, shortPath string) {
	p.sources = append(p.sources, fullPath)
	p.shortPaths = append(p.shortPaths, shortPath)
}

func (p Parser) Parse() ([]Message, error) {
	files, err := p.compiler.Compile(context.Background(), p.sources...)
	if err != nil {
		return nil, fmt.Errorf("couldn't compile: %w", err)
	}

	var messages []Message
	for i, file := range files {
		shortPath := p.shortPaths[i]
		for msgIndex := range file.Messages().Len() {
			messages = append(messages, Message{
				ShortPath:  shortPath,
				Descriptor: file.Messages().Get(msgIndex),
			})
		}
	}

	return messages, nil
}

type Message struct {
	ShortPath  string
	Descriptor protoreflect.MessageDescriptor
}
