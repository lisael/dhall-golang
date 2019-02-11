package ast

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func judgmentallyEqual(e1 Expr, e2 Expr) bool {
	// TODO: alpha-normalization
	ne1 := e1.Normalize()
	ne2 := e2.Normalize()
	return reflect.DeepEqual(ne1, ne2)
}

func (c Const) TypeWith(*TypeContext) (Expr, error) {
	if c == Type {
		return Kind, nil
	}
	if c == Kind {
		return Sort, nil
	}
	return nil, errors.New("Sort has no type")
}

func (v Var) TypeWith(ctx *TypeContext) (Expr, error) {
	if t, ok := ctx.Lookup(v.Name, 0); ok {
		return t, nil
	}
	return nil, fmt.Errorf("Unbound variable %s, context was %+v", v.Name, ctx)
}

func (lam *LambdaExpr) TypeWith(ctx *TypeContext) (Expr, error) {
	if _, err := lam.Type.TypeWith(ctx); err != nil {
		return nil, err
	}
	argType := lam.Type.Normalize()
	newctx := ctx.Insert(lam.Label, argType).Map(func(e Expr) Expr { return Shift(1, Var{Name: lam.Label}, e) })
	bodyType, err := lam.Body.TypeWith(newctx)
	if err != nil {
		return nil, err
	}

	p := &Pi{Label: lam.Label, Type: argType, Body: bodyType}
	_, err2 := p.TypeWith(ctx)
	if err2 != nil {
		return nil, err2
	}

	return p, nil
}

func (pi *Pi) TypeWith(ctx *TypeContext) (Expr, error) {
	argType, err := pi.Type.TypeWith(ctx)
	if err != nil {
		return nil, err
	}
	tA := argType.Normalize()
	// FIXME return error rather than panic if tA isn't a
	// Const
	kA := tA.(Const)
	// FIXME: proper de bruijn indices to avoid variable capture
	// FIXME: modifying context in place is.. icky
	(*ctx)[pi.Label] = append([]Expr{pi.Type.Normalize()}, (*ctx)[pi.Label]...)
	bodyType, err := pi.Body.TypeWith(ctx)
	if err != nil {
		return nil, err
	}
	tB := bodyType.Normalize()
	// FIXME return error rather than panic if tA isn't a
	// Const
	kB := tB.(Const)
	// Restore ctx to how it was before
	(*ctx)[pi.Label] = (*ctx)[pi.Label][1:len((*ctx)[pi.Label])]

	return Rule(kA, kB)
}

func (app *App) TypeWith(ctx *TypeContext) (Expr, error) {
	fnType, err := app.Fn.TypeWith(ctx)
	if err != nil {
		return nil, err
	}
	tF := fnType.Normalize()
	pF, ok := tF.(*Pi)
	if !ok {
		return nil, fmt.Errorf("Expected %s to be a function type", tF)
	}

	argType, err := app.Arg.TypeWith(ctx)
	if err != nil {
		return nil, err
	}
	// FIXME replace == with a JudgmentallyEqual() fn here
	if pF.Type == argType {
		a := Shift(1, Var{Name: pF.Label}, app.Arg)
		b := Subst(Var{Name: pF.Label}, a, pF.Body)
		return Shift(-1, Var{Name: pF.Label}, b), nil
	} else {
		return nil, errors.New("type mismatch between lambda and applied value")
	}
}

func (a Annot) TypeWith(ctx *TypeContext) (Expr, error) {
	_, err := a.Annotation.TypeWith(ctx)
	if err != nil {
		return nil, err
	}
	t2, err := a.Expr.TypeWith(ctx)
	if err != nil {
		return nil, err
	}
	if !judgmentallyEqual(a.Annotation, t2) {
		var b strings.Builder
		b.WriteString("Annotation mismatch: inferred type ")
		t2.WriteTo(&b)
		b.WriteString(" but annotated ")
		a.Annotation.WriteTo(&b)
		return nil, errors.New(b.String())
	}
	return t2, nil
}

func (double) TypeWith(*TypeContext) (Expr, error) { return Type, nil }

func (DoubleLit) TypeWith(*TypeContext) (Expr, error) { return Double, nil }

func (boolean) TypeWith(*TypeContext) (Expr, error) { return Type, nil }

func (BoolLit) TypeWith(*TypeContext) (Expr, error) { return Bool, nil }

func (natural) TypeWith(*TypeContext) (Expr, error) { return Type, nil }

func (NaturalLit) TypeWith(*TypeContext) (Expr, error) { return Natural, nil }

func (p NaturalPlus) TypeWith(ctx *TypeContext) (Expr, error) {
	L, err := p.L.TypeWith(ctx)
	if err != nil {
		return nil, err
	}
	L = L.Normalize()
	if L != Natural {
		return nil, fmt.Errorf("Expecting a Natural, can't add %s", L)
	}
	R, err := p.R.TypeWith(ctx)
	if err != nil {
		return nil, err
	}
	R = R.Normalize()
	if R != Natural {
		return nil, fmt.Errorf("Expecting a Natural, can't add %s", R)
	}
	return Natural, nil
}

func (integer) TypeWith(*TypeContext) (Expr, error) { return Type, nil }

func (IntegerLit) TypeWith(*TypeContext) (Expr, error) { return Integer, nil }

func (list) TypeWith(*TypeContext) (Expr, error) { return &Pi{"_", Type, Type}, nil }

func (l EmptyList) TypeWith(ctx *TypeContext) (Expr, error) {
	t := l.Type
	k, err := t.TypeWith(ctx)
	if err != nil {
		return nil, err
	}
	if k.Normalize() != Type {
		return nil, fmt.Errorf("List annotation %s is not a Type", t)
	}
	return &App{List, t}, nil
}

func (l NonEmptyList) TypeWith(ctx *TypeContext) (Expr, error) {
	exprs := []Expr(l)
	t, err := exprs[0].TypeWith(ctx)
	if err != nil {
		return nil, err
	}
	k, err := t.TypeWith(ctx)
	if k.Normalize() != Type {
		return nil, fmt.Errorf("Invalid type for List elements")
	}
	for _, elem := range exprs[1:] {
		t2, err := elem.TypeWith(ctx)
		if err != nil {
			return nil, err
		}
		if !judgmentallyEqual(t, t2) {
			return nil, fmt.Errorf("All List elements must have same type, but types %s and %s don't match", t, t2)
		}
	}
	return &App{List, t}, nil
}
