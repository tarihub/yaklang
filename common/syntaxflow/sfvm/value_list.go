package sfvm

import (
	"regexp"
	"strings"

	"github.com/yaklang/yaklang/common/go-funk"
	"github.com/yaklang/yaklang/common/utils"
)

var _ ValueOperator = &ValueList{}

func MergeValues(values ...ValueOperator) ValueOperator {
	return NewValues(values)
}

func NewValues(values []ValueOperator) ValueOperator {
	return &ValueList{values: values}
}

type ValueList struct {
	values []ValueOperator
}

func (v ValueList) GetCalled() (ValueOperator, error) {
	var res []ValueOperator
	for _, v := range v.values {
		called, err := v.GetCalled()
		if err != nil {
			continue
		}
		res = append(res, called)
	}
	return NewValues(res), nil
}

func (v ValueList) ForEach(h func(i any)) {
	funk.ForEach(v.values, func(i any) {
		h(i)
	})
}

func (v *ValueList) String() string {
	var res []string
	for _, v := range v.values {
		res = append(res, v.String())
	}
	return strings.Join(res, "; ")
}

func (v *ValueList) GetCallActualParams(i int) (ValueOperator, error) {
	var res []ValueOperator
	for _, v := range v.values {
		def, err := v.GetCallActualParams(i)
		if err != nil {
			return nil, err
		}
		res = append(res, def)
	}
	return NewValues(res), nil
}
func (v *ValueList) GetAllCallActualParams() (ValueOperator, error) {
	var res []ValueOperator
	for _, v := range v.values {
		def, err := v.GetAllCallActualParams()
		if err != nil {
			return nil, err
		}
		res = append(res, def)
	}
	return NewValues(res), nil
}

func (v *ValueList) GetSyntaxFlowDef() (ValueOperator, error) {
	var res []ValueOperator
	for _, v := range v.values {
		def, err := v.GetSyntaxFlowDef()
		if err != nil {
			return nil, err
		}
		res = append(res, def)
	}
	return NewValues(res), nil
}

func (v *ValueList) GetSyntaxFlowUse() (ValueOperator, error) {
	var res []ValueOperator
	for _, v := range v.values {
		use, err := v.GetSyntaxFlowUse()
		if err != nil {
			return nil, err
		}
		res = append(res, use)
	}
	return NewValues(res), nil
}
func (v *ValueList) GetSyntaxFlowTopDef(config ...*ConfigItem) (ValueOperator, error) {
	var res []ValueOperator
	for _, v := range v.values {
		topDef, err := v.GetSyntaxFlowTopDef(config...)
		if err != nil {
			return nil, err
		}
		res = append(res, topDef)
	}
	return NewValues(res), nil
}

func (v *ValueList) GetSyntaxFlowBottomUse(config ...*ConfigItem) (ValueOperator, error) {
	var res []ValueOperator
	for _, v := range v.values {
		bottomUse, err := v.GetSyntaxFlowBottomUse()
		if err != nil {
			return nil, err
		}
		res = append(res, bottomUse)
	}
	return NewValues(res), nil
}

func (v *ValueList) ListIndex(i int) (ValueOperator, error) {
	if i < 0 || i >= len(v.values) {
		return nil, utils.Error("index out of range")
	}
	return v.values[i], nil
}

func (v *ValueList) IsMap() bool {
	return false
}

func (v *ValueList) IsList() bool {
	return true
}

func (v *ValueList) ExactMatch(isMember bool, s string) (bool, ValueOperator, error) {
	var res []ValueOperator
	for _, value := range v.values {
		match, next, err := value.ExactMatch(isMember, s)
		if err != nil {
			return false, nil, err
		}
		if match {
			if next != nil {
				res = append(res, next)
			}
		}
	}
	return len(res) > 0, NewValues(res), nil
}

func (v *ValueList) GlobMatch(isMember bool, s Glob) (bool, ValueOperator, error) {
	var res []ValueOperator
	for _, value := range v.values {
		match, next, err := value.GlobMatch(isMember, s)
		if err != nil {
			return false, nil, err
		}
		if match {
			if next != nil {
				res = append(res, next)
			}
		}
	}
	return len(res) > 0, NewValues(res), nil
}

func (v *ValueList) RegexpMatch(isMember bool, regexp *regexp.Regexp) (bool, ValueOperator, error) {
	var res []ValueOperator
	for _, value := range v.values {
		match, next, err := value.RegexpMatch(isMember, regexp)
		if err != nil {
			return false, nil, err
		}
		if match {
			if next != nil {
				res = append(res, next)
			}
		}
	}
	return len(res) > 0, NewValues(res), nil
}
