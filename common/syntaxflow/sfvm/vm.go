package sfvm

import (
	"fmt"
	"sync"

	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/syntaxflow/sf"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/utils/omap"
	"github.com/yaklang/yaklang/common/yak/antlr4util"
)

type SyntaxFlowVirtualMachine struct {
	vars *omap.OrderedMap[string, ValueOperator]

	debug      bool
	frameMutex *sync.Mutex
	frames     []*SFFrame
}

func NewSyntaxFlowVirtualMachine() *SyntaxFlowVirtualMachine {
	sfv := &SyntaxFlowVirtualMachine{
		vars:       omap.NewEmptyOrderedMap[string, ValueOperator](),
		frameMutex: new(sync.Mutex),
	}
	return sfv
}

func (s *SyntaxFlowVirtualMachine) Debug(i ...bool) *SyntaxFlowVirtualMachine {
	if len(i) > 0 {
		s.debug = i[0]
	} else {
		s.debug = true
	}
	return s
}

func (s *SyntaxFlowVirtualMachine) Show() {
	for _, i := range s.frames {
		for _, c := range i.Codes {
			fmt.Println(c.String())
		}
	}
}

func (s *SyntaxFlowVirtualMachine) ForEachFrame(h func(frame *SFFrame)) {
	for _, i := range s.frames {
		h(i)
	}
}

func (s *SyntaxFlowVirtualMachine) Compile(text string) (ret error) {
	if text == "" {
		return utils.Errorf("SyntaxFlow compile error: text is nil")
	}
	defer func() {
		if err := recover(); err != nil {
			ret = utils.Wrapf(utils.Error(err), "Panic for SyntaxFlow compile")
		}
	}()
	errLis := antlr4util.NewErrorListener()

	lexer := sf.NewSyntaxFlowLexer(antlr.NewInputStream(text))
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(errLis)
	astParser := sf.NewSyntaxFlowParser(antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel))
	astParser.RemoveErrorListeners()
	astParser.AddErrorListener(errLis)

	result := NewSyntaxFlowVisitor()
	flow := astParser.Flow()
	// fmt.Printf("%v\n", flow.ToStringTree(nil, astParser))
	if len(errLis.GetErrors()) > 0 {
		return utils.Errorf("SyntaxFlow compile error: %v", errLis.GetErrorString())
	}

	result.text = text
	result.VisitFlow(flow)
	var frame = result.CreateFrame(s.vars)
	s.frames = append(s.frames, frame)

	return nil
}

func (s *SyntaxFlowVirtualMachine) Feed(i ValueOperator) *omap.OrderedMap[string, ValueOperator] {
	s.frameMutex.Lock()
	defer s.frameMutex.Unlock()

	result := omap.NewOrderedMap(map[string]ValueOperator{})
	for index, frame := range s.frames {
		err := frame.Debug(s.debug).exec(i)
		if err != nil {
			log.Errorf("exec frame[%v]: %v\n\t\tCODE: %v", err, index, frame.Text)
		}
		if frame.stack.Len() > 0 {
			log.Infof("stack unbalanced: %v", frame.stack.Len())
		}
	}
	s.vars.Map(func(s string, a ValueOperator) (string, ValueOperator, error) {
		result.Set(s, a)
		return s, a, nil
	})
	return result
}
