package ssa

import (
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/yaklang/yaklang/common/utils/memedit"
)

func (b *FunctionBuilder) SetRangeFromTerminalNode(node antlr.TerminalNode) func() {
	return b.SetRange(NewToken(node))
}
func (b *FunctionBuilder) SetRange(token CanStartStopToken) func() {
	r := GetRange(b.GetEditor(), token)
	if r == nil {
		return func() {}
	}
	backup := b.CurrentRange
	b.CurrentRange = r

	return func() {
		b.CurrentRange = backup
	}
}

// / ============================== Token ==============================
type CanStartStopToken interface {
	GetStop() antlr.Token
	GetStart() antlr.Token
	GetText() string
}

func GetEndPosition(t antlr.Token) (int, int) {
	var line, column int
	str := strings.Split(t.GetText(), "\n")
	if len(str) > 1 {
		line = t.GetLine() + len(str) - 1
		column = len(str[len(str)-1])
	} else {
		line = t.GetLine()
		column = t.GetColumn() + len(str[0])
	}
	return line, column
}

func GetRange(editor *memedit.MemEditor, token CanStartStopToken) *Range {
	startToken := token.GetStart()
	endToken := token.GetStop()
	if startToken == nil || endToken == nil {
		return nil
	}

	start := NewPosition(int64(startToken.GetLine()), int64(startToken.GetColumn()))

	endLine, endColumn := GetEndPosition(endToken)
	end := NewPosition(int64(endLine), int64(endColumn))
	return NewRange(editor, start, end)
}

type Token struct {
	start antlr.Token
	end   antlr.Token
	text  string
}

func NewToken(node antlr.TerminalNode) *Token {
	return &Token{
		start: node.GetSymbol(),
		end:   node.GetSymbol(),
		text:  node.GetText(),
	}
}

func (t *Token) GetStart() antlr.Token {
	return t.start
}
func (t *Token) GetStop() antlr.Token {
	return t.end
}
func (t *Token) GetText() string {
	return t.text
}

var _ CanStartStopToken = (*Token)(nil)
