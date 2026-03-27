package funcs

import (
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/elliotchance/pie/pie"
)

// expression     → equality ;
// releation      → equality  ( ( "AND" | "OR" ) equality )*
// equality       → comparison ( ( "!=" | "==" ) comparison )* ;
// comparison     → addition ( ( ">" | ">=" | "<" | "<=" ) addition )* ;
// addition       → multiplication ( ( "-" | "+" ) multiplication )* ;
// multiplication → unary ( ( "/" | "*" ) unary )* ;
// unary          → ( "!" | "-" ) unary
//                | primary ;
// primary        → NUMBER | STRING | "false" | "true" | "nil"
//                | "(" expression ")" ;

//HandlerFunc is for
type HandlerFunc func(params ...interface{}) interface{}

// Expression is for
type Expression struct {
	//Releation is for
	Releation *Releation `@@`
}

// Releation is for
type Releation struct {
	Equality *Equality  `@@`
	Op       string     `[ @( "&" "&" | "|" "|" )`
	Next     *Releation `  @@ ]`
}

//Eval is for
func (rele *Releation) Eval(ctx *EvalContext) interface{} {
	l := rele.Equality.Eval(ctx)
	if rele.Op == "" {
		return l
	}
	handler, ok := ctx.Funcs[rele.Op]
	if ok == false {
		return fmt.Errorf("no Releation handler for : " + rele.Op)
	}
	r := rele.Next.Eval(ctx)
	return handler(l, r)

}

// Equality is for
type Equality struct {
	Comparison *Comparison `@@`
	Op         string      `[ @( "!" "=" | "=" "=" )`
	Next       *Equality   `  @@ ]`
}

//Eval is for
func (equal *Equality) Eval(ctx *EvalContext) interface{} {
	l := equal.Comparison.Eval(ctx) // left
	if equal.Op == "" {
		return l //left
	}
	handler, ok := ctx.Funcs[equal.Op]
	if ok == false {
		return fmt.Errorf("no Multiplication handler for : " + equal.Op)
	}

	r := equal.Next.Eval(ctx)
	return handler(l, r)
}

// Comparison is for
type Comparison struct {
	Addition *Addition   `@@`
	Op       string      `[ @( ">" | ">" "=" | "<" | "<" "=" )`
	Next     *Comparison `  @@ ]`
}

// Eval is for
func (comp *Comparison) Eval(ctx *EvalContext) interface{} {
	if comp.Op == "" {
		return comp.Addition.Eval(ctx)
	}
	handler, ok := ctx.Funcs[comp.Op]
	if ok == false {
		return fmt.Errorf("no Comparison handler for : " + comp.Op)
	}
	return handler(comp.Addition.Eval(ctx), comp.Next.Eval(ctx))
}

// AddTerm is for
type AddTerm struct {
	Op   string    `parser:"@( '-' | '+' )"`
	Next *Addition `parser:" @@"`
}

// Eval is for
func (addterm *AddTerm) Eval(ctx *EvalContext, l interface{}) interface{} {
	handler, ok := ctx.Funcs[addterm.Op]
	if ok == false {
		return fmt.Errorf("no Multiplication handler for : " + addterm.Op)
	}
	r := addterm.Next.Eval(ctx)
	return handler(l, r)
}

// Addition is for
type Addition struct {
	Multiplication *Multiplication `parser:"@@"`
	ATerms         []*AddTerm      `parser:" @@*"`
}

// Eval is for
func (addt *Addition) Eval(ctx *EvalContext) interface{} {
	l := addt.Multiplication.Eval(ctx)
	if len(addt.ATerms) == 0 {
		return l
	}

	for _, term := range addt.ATerms {
		l = term.Eval(ctx, l)
	}
	return l
}

// MTerm is for
type MTerm struct {
	Op   string `parser:"@( '/' | '*' )"`
	Next *Unary `parser:" @@ "`
}

// Eval is for
func (mterm *MTerm) Eval(ctx *EvalContext, l interface{}) interface{} {
	handler, ok := ctx.Funcs[mterm.Op]
	if ok == false {
		return fmt.Errorf("no Multiplication handler for : " + mterm.Op)
	}
	r := mterm.Next.Eval(ctx)
	return handler(l, r)
}

// Multiplication is for
type Multiplication struct {
	Unary  *Unary   `parser:"@@"`
	MTerms []*MTerm `parser:"@@*"`
}

//Eval is for
func (mult *Multiplication) Eval(ctx *EvalContext) interface{} {
	l := mult.Unary.Eval(ctx)
	if len(mult.MTerms) == 0 {
		return l
	}

	for _, term := range mult.MTerms {
		l = term.Eval(ctx, l)
	}
	return l
	//return nil
}

// Eval is for Unary
func (unary *Unary) Eval(ctx *EvalContext) interface{} {
	// func handle
	handler, ok := ctx.Funcs[unary.Op]
	if ok == false {
		return unary.Primary.Eval(ctx)
	}
	if unary.Unary != nil {
		return handler(unary.Unary.Eval(ctx))
	}
	return fmt.Errorf("no Unary handler for : " + unary.Op)
}

// Unary is for
type Unary struct {
	Op      string   `  ( @( "!" | "-" )`
	Unary   *Unary   `    @@ )`
	Primary *Primary `| @@`
}

// Eval is for
func (prim *Primary) Eval(ctx *EvalContext) interface{} {

	if prim.Number != nil {
		return *prim.Number
	}
	if prim.String != nil {
		return *prim.String
	}
	if prim.SubExpression != nil {
		return prim.SubExpression.Eval(ctx)
	}
	if prim.Var != nil {
		return prim.Var.Eval(ctx)
	}
	return fmt.Errorf("not supported")
}

type FuncField struct {
	Name      string      `@Ident`
	Arguments []*Argument `( "(" ( @@ ( "," @@ )* )? ")" )?`
}

//Eval is for
func (arg *Argument) Eval(ctx *EvalContext) interface{} {
	if arg.Expr == nil {
		fmt.Errorf("Expr not error")
	}
	return arg.Expr.Eval(ctx)
}

//Primary is for
type Primary struct {
	Number        *float64    `  @Float | @Int`
	String        *string     `| @String`
	Bool          *bool       `| ( @"true" | "false" )`
	Nil           bool        `| @"nil"`
	SubExpression *Expression `| "(" @@ ")" `
	Var           *FuncField  `| @@ `
}

type Argument struct {
	Expr *Expression `@@`
}

//Stmt is for
type Stmt struct {
	ValDef *string `@Ident "="`
	//Field  *FuncField  `(@@ `
	Expr *Expression `@@` //`| @@)`
}

//Eval is for
func (expr *Expression) Eval(ctx *EvalContext) interface{} {
	//if expr.Equality == nil {
	//	return fmt.Errorf("no Expression ")
	//}
	//return expr.Equality.Eval(ctx)
	if expr.Releation == nil {
		return fmt.Errorf("no Expression ")
	}
	return expr.Releation.Eval(ctx)
}

// Eval is for
func (fun *FuncField) Eval(ctx *EvalContext) interface{} {

	if len(fun.Arguments) == 0 {
		val, ok := ctx.Values[fun.Name]
		if ok == true {
			return val
		}
	}
	handler, ok := ctx.Funcs[fun.Name]

	if ok == false {
		return pie.Float64s{}
	}
	length := len(fun.Arguments)

	s := make([]interface{}, length)
	index := 0
	for i := length - 1; i >= 0; i-- {
		s[i] = fun.Arguments[i].Eval(ctx)
		//fmt.Println("fun.Arguments", s[i])
		index++
	}
	return handler(s...)
}

// Eval is for
func (stmt *Stmt) Eval(ctx *EvalContext) {
	if stmt.ValDef != nil {
		//if stmt.Field != nil {
		//	val := stmt.Field.Eval(ctx)    //left
		//	ctx.Values[*stmt.ValDef] = val // right
		//} else if stmt.Expr != nil {
		val := stmt.Expr.Eval(ctx)     //left
		ctx.Values[*stmt.ValDef] = val // right
		//}
	}
}

// PStmts is for
type PStmts struct {
	Stmts []*Stmt `@@*`
}

// Eval is for
func (pstmt *PStmts) Eval(ctx *EvalContext) {
	for _, stmt := range pstmt.Stmts {
		stmt.Eval(ctx)
	}
}

// Compile is for
func Compile() *participle.Parser {
	parser := participle.MustBuild(&PStmts{}, participle.UseLookahead(2)) //participle.UseLookahead(2))
	return parser
}
