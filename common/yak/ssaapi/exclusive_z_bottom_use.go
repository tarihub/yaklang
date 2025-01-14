package ssaapi

import (
	"sort"

	"github.com/samber/lo"
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/utils/omap"
	"github.com/yaklang/yaklang/common/yak/ssa"
)

func (v *Value) GetBottomUses(opt ...OperationOption) Values {
	actx := NewAnalyzeContext(opt...)
	actx.Self = v
	ret := v.getBottomUses(actx)
	lo.UniqBy(ret, func(item *Value) int64 {
		return item.GetId()
	})
	return ret
}

func (v *Value) visitUserFallback(actx *AnalyzeContext) Values {
	var vals Values
	v.GetUsers().ForEach(func(value *Value) {
		if ret := value.AppendDependOn(v).getBottomUses(actx); len(ret) > 0 {
			vals = append(vals, ret...)
		}
	})

	// member.IsUndefined()
	undefineMember := false
	if un, ok := ssa.ToUndefined(v.node); ok {
		if un.Kind == ssa.UndefinedMemberInValid || un.Kind == ssa.UndefinedMemberValid {
			undefineMember = true
		}
	}
	if v.IsMember() && !undefineMember && actx.TheMemberShouldBeVisited(v) {
		obj := v.GetObject()
		actx.PushObject(obj, v.GetKey(), v)
		vals = append(vals, obj.getBottomUses(actx)...)
		actx.PopObject()
	}
	if len(vals) <= 0 {
		return Values{v}
	}
	return vals
}

func (v *Value) getBottomUses(actx *AnalyzeContext, opt ...OperationOption) Values {
	if actx == nil {
		actx = NewAnalyzeContext(opt...)
	}

	actx.depth++
	defer func() {
		actx.depth--
	}()
	v.SetDepth(actx.depth)
	if actx.config.MaxDepth > 0 && actx.depth > actx.config.MaxDepth {
		return Values{}
	}
	if actx.config.MinDepth < 0 && actx.depth < actx.config.MinDepth {
		return Values{}
	}

	if actx.config.HookEveryNode != nil {
		err := actx.config.HookEveryNode(v)
		if err != nil {
			log.Errorf("hook every node failed: %v", err)
		}
	}

	if ValueCompare(v, actx.Self) {
		return v.visitUserFallback(actx)
	}

	switch ins := v.node.(type) {
	case *ssa.LazyInstruction:
		v.node = ins.Self()
		return v.getBottomUses(actx, opt...)
	case *ssa.Phi:
		// enter function via phi
		if !actx.ThePhiShouldBeVisited(v) {
			// the phi is existed, visited in the same stack.
			return Values{}
		}
		actx.VisitPhi(v)
		return v.visitUserFallback(actx)
	case *ssa.Call:
		if !actx.TheCallShouldBeVisited(v) {
			// call existed
			return v.visitUserFallback(actx)
		}

		if ins.Method == nil {
			// log.Infof("fallback: (call instruction 's method/func is not *Function) unknown caller, got: %v", ins.Method.String())
			return v.visitUserFallback(actx)
		}

		// enter function via call
		f, ok := ssa.ToFunction(ins.Method)
		if !ok {
			log.Infof("fallback: (call instruction 's method/func is not *Function) unknown caller, got: %v", ins.Method.String())
			return v.visitUserFallback(actx)
		}

		funcValue := NewValue(f).AppendDependOn(v)
		if ValueCompare(funcValue, actx.Self) {
			return v.visitUserFallback(actx)
		}

		// push call
		err := actx.PushCall(v)
		if err != nil {
			log.Infof("push call failed: %v", err)
			return v.visitUserFallback(actx)
			// existed call
		} else {
			defer actx.PopCall()
		}

		// try to find formal param index from call
		// v is calling instruction
		// funcValue is the function
		existed := map[int64]struct{}{}
		v.DependOn.ForEach(func(value *Value) {
			existed[value.GetId()] = struct{}{}
		})
		var formalParamsIndex = make([]int, 0, len(ins.Args))
		for argIndex, targetIndex := range ins.Args {
			if _, ok := existed[targetIndex.GetId()]; ok {
				formalParamsIndex = append(formalParamsIndex, argIndex)
			}
		}
		var params = omap.NewOrderedMap(map[int64]*ssa.Parameter{})
		lo.ForEach(f.Param, func(param *ssa.Parameter, index int) {
			for _, i := range formalParamsIndex {
				if index == i {
					params.Set(param.GetId(), param)
				}
			}
		})
		if lo.Max(formalParamsIndex) >= len(f.Param) && len(f.Param) > 0 {
			last, _ := lo.Last(f.Param)
			if last != nil {
				params.Set(last.GetId(), last)
			}
		}

		var vals Values
		if params.Len() > 0 {
			for _, formalParam := range params.Values() {
				rets := NewValue(formalParam).AppendDependOn(funcValue).getBottomUses(actx, opt...)
				vals = append(vals, rets...)
			}
			return vals
		}

		// no formal parameters found!
		// enter return
		for _, retStmt := range f.Return {
			retVals := NewValue(retStmt).AppendDependOn(funcValue)
			vals = append(vals, retVals)
		}
		return vals
	case *ssa.Return:
		// enter function via return
		fallback := func() Values {
			// var results Values
			results := make(Values, 0)
			if f := ins.GetFunc(); f != nil {
				NewValue(f).GetCalledBy().ForEach(func(value *Value) {
					dep := value.AppendDependOn(v)
					err := actx.PushCall(dep)

					if err != nil {
						log.Errorf("push call failed: %v", err)
					} else {
						defer actx.PopCall()
					}
					results = append(results, dep.getBottomUses(actx)...)
				})
			}
			if len(results) > 0 {
				return results
			}
			for _, result := range ins.Results {
				results = append(results, NewValue(result).AppendDependOn(v))
			}
			return results
		}
		if actx._callStack.Len() > 0 {
			existed := make(map[int64]struct{})
			v.DependOn.ForEach(func(value *Value) {
				existedId := value.GetId()
				existed[existedId] = struct{}{}
			})
			var indexes = make(map[int]struct{})
			for idx, ret := range ins.Results {
				if _, ok := existed[ret.GetId()]; ok {
					indexes[idx] = struct{}{}
				}
			}

			currentCallValue := actx.GetCurrentCall()
			if currentCallValue == nil {
				return fallback()
			}
			call, ok := ssa.ToCall(currentCallValue.node)
			fun, ok := ssa.ToFunction(call.Method)
			if !ok {
				log.Warnf("BUG: (call's fun is not clean!) unknown function: %v", v.String())
				return fallback()
			}
			_ = fun //TODO: fun can tell u, which return value is the target

			var vals Values
			if !call.IsObject() || len(indexes) <= 0 {
				NewValue(call).GetUsers().ForEach(func(user *Value) {
					if ret := user.AppendDependOn(currentCallValue).AppendDependOn(v).getBottomUses(actx); len(ret) > 0 {
						vals = append(vals, ret...)
					}
				})

				if len(vals) > 0 {
					return vals
				}
				return NewValue(call).AppendDependOn(v).getBottomUses(actx)
			}

			// handle indexed return to call return
			orderedIndex := lo.Keys(indexes)
			sort.Ints(orderedIndex)
			for _, idx := range orderedIndex {
				indexedReturn, ok := call.GetIndexMember(idx)
				if !ok {
					continue
				}
				returnReceiver := NewValue(indexedReturn)
				actx.PushObject(currentCallValue, returnReceiver.GetKey(), returnReceiver)
				if newVals := returnReceiver.AppendDependOn(returnReceiver).AppendDependOn(v).getBottomUses(actx); len(newVals) > 0 {
					vals = append(vals, newVals...)
				}
				actx.PopObject()
			}
			if len(vals) > 0 {
				return vals
			}
			return NewValue(call).AppendDependOn(v).getBottomUses(actx)
		}
		return fallback()
	}
	return v.visitUserFallback(actx)
}
