package ssautil

type LabelTarget[T versionedValue] interface {
	Break(from ScopedVersionedTableIF[T])
	Continue(from ScopedVersionedTableIF[T])
	FallThough(from ScopedVersionedTableIF[T])
}

// BuildSyntaxBlock builds a syntax block using the provided scope and buildBody function.
/*
if this scope finish this program

* BuildBody should return true

* this function will return true
*/
func BuildSyntaxBlock[T versionedValue](
	global ScopedVersionedTableIF[T],
	buildBody func(ScopedVersionedTableIF[T]) ScopedVersionedTableIF[T],
) ScopedVersionedTableIF[T] {
	/*
		scope
			sub // build body
				--- body
			end // cover by body
	*/

	body := global.CreateSubScope()
	bodyEnd := buildBody(body)

	end := global.CreateSubScope()
	global.ForEachCapturedVariable(func(s string, vi VersionedIF[T]) {
		end.SetCapturedVariable(s, vi)
	})
	end.CoverBy(bodyEnd)
	return end
}

// IfStmt represents an if statement.
type IfStmt[T versionedValue] struct {
	global             ScopedVersionedTableIF[T]
	lastConditionScope ScopedVersionedTableIF[T]
	BodyScopes         []ScopedVersionedTableIF[T]
	hasElse            bool
}

// NewIfStmt creates a new IfStmt with the given global scope.
/*
	IfStmt will handle if-stmt scope.
	API:
		* BuildItem(condition fun(scope), body func(scope)):
			build if item using the provided Condition and Body functions.
		* BuildElse(elseBody func(scope)):
			set the else function for the IfStmt.
		* BuildFinish(mergeHandler func(name string, t []T) T):
			build the IfStmt finish, using the provided mergeHandler function create Phi.
	IfStmt will build this scope when this method call
*/
func NewIfStmt[T versionedValue](global ScopedVersionedTableIF[T]) *IfStmt[T] {
	// condition := global.CreateSubScope()
	return &IfStmt[T]{
		global:             global,
		lastConditionScope: global,
		BodyScopes:         make([]ScopedVersionedTableIF[T], 0),
		hasElse:            false,
	}
}

// BuildItem build the if item using the provided Condition and Body functions.
func (i *IfStmt[T]) BuildItem(Condition func(ScopedVersionedTableIF[T]), Body func(ScopedVersionedTableIF[T]) ScopedVersionedTableIF[T]) {
	if i.hasElse {
		panic("cannot add item after else")
	}

	// create new condition and body scope
	i.lastConditionScope = i.lastConditionScope.CreateSubScope()
	Condition(i.lastConditionScope)

	bodyScope := i.lastConditionScope.CreateSubScope()
	end := Body(bodyScope)
	if end != nil {
		i.BodyScopes = append(i.BodyScopes, end)
	}
}

// SetElse sets the else function for the IfStmt.
func (i *IfStmt[T]) BuildElse(elseBody func(ScopedVersionedTableIF[T]) ScopedVersionedTableIF[T]) {
	elseScope := i.lastConditionScope.CreateSubScope()
	end := elseBody(elseScope)
	if end != nil {
		i.BodyScopes = append(i.BodyScopes, end)
	}
	i.hasElse = true
}

// Build builds the IfStmt using the provided mergeHandler function.
func (i *IfStmt[T]) BuildFinish(
	mergeHandler MergeHandle[T],
) ScopedVersionedTableIF[T] {
	/*
		global
			condition1 // condition
				body1 // body
				condition2 // condition
					body2 // body
					...
					else // else // same level with last body
		end // end scope
		// [phi] from all body and else
	*/

	endScope := i.global.CreateSubScope()

	endScope.Merge(
		!i.hasElse, // has base
		mergeHandler,
		i.BodyScopes...,
	)
	return endScope
}

// LoopStmt represents a loop statement.
type LoopStmt[T versionedValue] struct {
	MergeToEnd   []ScopedVersionedTableIF[T] // break, merge phi in exit
	MergeToLatch []ScopedVersionedTableIF[T] // continue, merge phi in latch

	ThirdBuilder func(ScopedVersionedTableIF[T]) // third

	global    ScopedVersionedTableIF[T]
	header    ScopedVersionedTableIF[T]
	condition ScopedVersionedTableIF[T]
	body      ScopedVersionedTableIF[T]
}

// NoneBuilder is a helper function that does nothing.
// func NoneBuilder[T comparable](ScopedVersionedTableIF[T])                                     {}
// func NoneBuilderReturnScope[T comparable](ScopedVersionedTableIF[T]) ScopedVersionedTableIF[T] {}

// NewLoopStmt creates a new LoopStmt with the given global scope.
func NewLoopStmt[T versionedValue](global ScopedVersionedTableIF[T], NewPhi func(string) T) *LoopStmt[T] {
	l := &LoopStmt[T]{
		global: global,
	}
	l.header = l.global.CreateSubScope()
	l.condition = l.header.CreateSubScope()
	l.condition.SetSpin(NewPhi)
	l.body = l.condition.CreateSubScope()
	l.ThirdBuilder = nil
	return l
}

// SetFirst sets the first function for the LoopStmt.
func (l *LoopStmt[T]) SetFirst(f func(ScopedVersionedTableIF[T])) {
	f(l.header)
}

// SetCondition sets the condition function for the LoopStmt.
func (l *LoopStmt[T]) SetCondition(f func(ScopedVersionedTableIF[T])) {
	f(l.condition)
}

// SetThird sets the third function for the LoopStmt.
func (l *LoopStmt[T]) SetThird(f func(ScopedVersionedTableIF[T])) {
	l.ThirdBuilder = f
}

// SetBody sets the body function for the LoopStmt.
func (l *LoopStmt[T]) SetBody(f func(ScopedVersionedTableIF[T]) ScopedVersionedTableIF[T]) {
	l.body = f(l.body)
}

func (l *LoopStmt[T]) Continue(from ScopedVersionedTableIF[T]) {
	l.MergeToLatch = append(l.MergeToLatch, from)
}

func (l *LoopStmt[T]) Break(from ScopedVersionedTableIF[T]) {
	l.MergeToEnd = append(l.MergeToEnd, from)
}

func (l *LoopStmt[T]) FallThough(from ScopedVersionedTableIF[T]) {
	// do nothing
}

// Build builds the LoopStmt using the provided NewPhi and SpinHandler functions.
func (l *LoopStmt[T]) Build(
	SpinHandler SpinHandle[T],
	mergeLatch MergeHandle[T],
	mergeEnd MergeHandle[T],
) ScopedVersionedTableIF[T] {

	/*
		global [i = 0]
			header [i] // first
				condition // condition [phi] from header and latch
					body [i] // body
						latch    // third [phi] from all continue and body
			exit // exit loop [phi]  from all break and global

		// in body
		* break to global scope
		* continue to latch scope
	*/

	// latch
	latch := l.body.CreateSubScope()
	latch.Merge(
		true,
		mergeLatch,
		l.MergeToLatch...,
	)
	// this `l.ThirdBuilder` only set in `l.SetThird`
	if l.ThirdBuilder != nil {
		// if not nil, mean, this `SetThird` is called before `SetBody`
		// call it
		l.ThirdBuilder(latch)
	}

	l.condition.Spin(l.header, latch, SpinHandler)

	// end
	end := l.global.CreateSubScope()
	l.header.CoverBy(l.condition)
	end.CoverBy(l.header)

	end.Merge(
		true,
		mergeEnd,
		l.MergeToEnd...,
	)

	return end
}

type TryStmt[T versionedValue] struct {
	global       ScopedVersionedTableIF[T]
	tryBody      ScopedVersionedTableIF[T]
	cacheBody    ScopedVersionedTableIF[T]
	finalBody    ScopedVersionedTableIF[T]
	ErrorName    string
	mergeHandler MergeHandle[T]
}

func NewTryStmt[T versionedValue](
	global ScopedVersionedTableIF[T],
	mergeHandler MergeHandle[T],
) *TryStmt[T] {
	return &TryStmt[T]{
		global:       global,
		mergeHandler: mergeHandler,
	}
}

func (t *TryStmt[T]) SetTryBody(body func(ScopedVersionedTableIF[T]) ScopedVersionedTableIF[T]) {
	tryBody := t.global.CreateSubScope()
	ret := body(tryBody)
	t.tryBody = ret

}

func (t *TryStmt[T]) SetError(name string) {
	t.ErrorName = name
}

func (t *TryStmt[T]) CreateCatch() ScopedVersionedTableIF[T] {
	t.cacheBody = t.global.CreateSubScope()
	return t.cacheBody
}

func (t *TryStmt[T]) SetCache(build func() ScopedVersionedTableIF[T]) {
	t.cacheBody.Merge(
		true,
		t.mergeHandler,
		t.tryBody,
	)
	t.cacheBody.CreateVariable(t.ErrorName, true)
	t.cacheBody = build()
}

func (t *TryStmt[T]) CreateFinally() ScopedVersionedTableIF[T] {
	t.finalBody = t.global.CreateSubScope()
	return t.finalBody
}

func (t *TryStmt[T]) SetFinal(build func() ScopedVersionedTableIF[T]) {
	t.finalBody.Merge(
		false, t.mergeHandler,
		t.tryBody, t.cacheBody,
	)
	ret := build()
	t.finalBody = ret
}

func (t *TryStmt[T]) Build() ScopedVersionedTableIF[T] {
	/*
		global
			try
				body
			catch
				...
			finally // option
				...
		end
	*/
	end := t.global.CreateSubScope()
	if t.finalBody != nil {
		end.CoverBy(t.finalBody)
	} else {
		end.Merge(
			false, t.mergeHandler,
			t.tryBody, t.cacheBody,
		)
	}
	return end
}

type SwitchStmt[T versionedValue] struct {
	global           ScopedVersionedTableIF[T]
	mergeToSwitchEnd []ScopedVersionedTableIF[T]
	mergeToNextBody  ScopedVersionedTableIF[T]
	AutoBreak        bool
}

func NewSwitchStmt[T versionedValue](global ScopedVersionedTableIF[T], AutoBreak bool) *SwitchStmt[T] {
	return &SwitchStmt[T]{
		global:    global,
		AutoBreak: AutoBreak,
	}
}

func (s *SwitchStmt[T]) Break(from ScopedVersionedTableIF[T]) {
	// do nothing
	s.mergeToSwitchEnd = append(s.mergeToSwitchEnd, from)
}

func (s *SwitchStmt[T]) Continue(from ScopedVersionedTableIF[T]) {
	// do nothing
}

func (s *SwitchStmt[T]) FallThough(from ScopedVersionedTableIF[T]) {
	// do nothing
	s.mergeToNextBody = from
}

func (s *SwitchStmt[T]) BuildBody(
	body func(ScopedVersionedTableIF[T]) ScopedVersionedTableIF[T],
	merge func(string, []T) T,
) {
	sub := s.global.CreateSubScope()
	if s.mergeToNextBody != nil {
		sub.Merge(true, merge, s.mergeToNextBody)
		s.mergeToNextBody = nil
	}
	ret := body(sub)
	if s.AutoBreak { // if this switch fall through, then merge to next body
		// if switch default break to switch end
		if s.mergeToNextBody == nil {
			// if not write FallThough, then merge to switch end
			s.mergeToSwitchEnd = append(s.mergeToSwitchEnd, ret)
		}
	} else {
		length := len(s.mergeToSwitchEnd)
		if length == 0 || s.mergeToSwitchEnd[length-1] != ret {
			s.mergeToNextBody = ret
		}
	}
}

func (s *SwitchStmt[T]) Build(merge func(string, []T) T) ScopedVersionedTableIF[T] {
	end := s.global.CreateSubScope()
	if s.AutoBreak {
		// if switch default break to switch end
		// just merge
		end.Merge(
			false,
			merge,
			s.mergeToSwitchEnd...,
		)
	} else {
		DefaultBody := s.mergeToNextBody
		if DefaultBody != nil {
			s.mergeToSwitchEnd = append(s.mergeToSwitchEnd, DefaultBody)
		}
		switch len(s.mergeToSwitchEnd) {
		case 0:
		case 1:
			// has default, no break, just cover by default
			end.CoverBy(s.mergeToSwitchEnd[0])
		default:
			end.Merge(false, merge, s.mergeToSwitchEnd...)
		}
	}
	return end
}
