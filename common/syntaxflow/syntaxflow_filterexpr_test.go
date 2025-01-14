package syntaxflow

import (
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/yaklang/yaklang/common/syntaxflow/sfvm"
)

func TestSyntaxFlowFilter_Search(t *testing.T) {
	for _, i := range []map[string]string{
		{
			"code":    "exec", // Ref
			"keyword": "push$exact",
		},
		{
			"code":    "ex*",
			"keyword": "push$glob",
		},
		{
			"code":    "/abc/",
			"keyword": "push$regex",
		},
	} {
		vm := sfvm.NewSyntaxFlowVirtualMachine()
		vm.Debug()
		err := vm.Compile(i["code"])
		if err != nil {
			t.Fatal(err)
		}
		vm.Show()
		count := 0
		checked := false
		vm.ForEachFrame(func(frame *sfvm.SFFrame) {
			count += len(frame.Codes)
			for _, c := range frame.Codes {
				if strings.Contains(c.String(), i["keyword"]) {
					checked = true
				}
			}
		})
		if !checked {
			t.Fatalf("SyntaxFlowVirtualMachine.Compile failed: %v", spew.Sdump(i))
		}
	}

}
