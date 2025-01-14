package ssa

import (
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/utils"
)

// ParseClassBluePrint  parse get classBluePrint if the ObjectType is a ClassFactor
func ParseClassBluePrint(this Value, objectTyp *ObjectType) (ret Type) {
	ret = objectTyp

	if !this.IsObject() {
		return
	}
	blue := NewClassBluePrint()
	// blue.SetObjectType(objectTyp)

	for key, member := range this.GetAllMember() {
		// if not function , just append this field to normal field
		typ := member.GetType()
		if typ.GetTypeKind() != FunctionTypeKind {
			// blue.NormalMember[key.String()] = member
			blue.AddNormalMember(key.String(), member)
			continue
		}

		fun := member.(*Function)
		if len(fun.Param) > 0 {
			if para := fun.Param[0]; para != nil && (para.IsObject() || para.HasUsers()) {
				blue.AddMethod(key.String(), fun)
				continue
			}
		}

		blue.AddNormalMember(key.String(), member)
	}

	if len(blue.GetMethod()) != 0 {
		return blue
	}

	return
}

func (c *ClassBluePrint) Apply(obj Value) Type {
	builder := obj.GetFunc().builder
	_ = builder

	call, isCall := ToCall(obj)

	objTyp := NewObjectType()
	objTyp.SetName(c.Name)
	objTyp.SetMethod(c.GetMethod())
	for _, parent := range c.ParentClass {
		parentObjectType := parent.Apply(obj)
		for key, f := range parentObjectType.GetMethod() {
			objTyp.AddMethod(key, f)
		}
	}

	for rawKey, member := range c.NormalMember {
		typ := member.Type
		value := member.Value
		key := builder.EmitConstInst(rawKey)
		log.Infof("apply key: %s, member: %v", key, member)

		objTyp.AddField(key, typ)

		// if in yaklang code, classBluePrint only create by function,
		// and Apply only called by function'call (call instruction)
		// and only this language, member can be set by `Parameter`,
		// we just create side-effect
		if para, ok := ToParameter(value); ok && isCall {
			sideEffect := builder.EmitSideEffect(key.String(), call, para)
			builder.AssignVariable(
				builder.CreateMemberCallVariable(obj, key),
				sideEffect,
			)
			continue
		}

		// in other language supported class,
		// classBluePrint only create by `class` keyword.
		// in this case, member can be set nil, just declare the type.
		if utils.IsNil(value) {
			value := builder.ReadMemberCallVariable(obj, key)
			value.SetType(typ)
		} else {
			builder.AssignVariable(
				builder.CreateMemberCallVariable(obj, key),
				value,
			)
		}
	}

	return objTyp
}
