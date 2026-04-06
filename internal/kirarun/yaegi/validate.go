package yaegi

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"

	"kira/kirarun"
)

// ValidateWorkflow checks that path is a loadable main package with a valid Run entrypoint
// for this kira version (Yaegi parse/compile + AST signature check). Does not execute Run.
func ValidateWorkflow(workflowPath string) error {
	abs, err := filepath.Abs(workflowPath)
	if err != nil {
		return fmt.Errorf("workflow path: %w", err)
	}
	if !filepath.IsAbs(abs) {
		return fmt.Errorf("workflow path must be absolute")
	}
	st, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("workflow file: %w", err)
	}
	if st.IsDir() {
		return fmt.Errorf("workflow path must be a file")
	}

	src, err := os.ReadFile(abs) // #nosec G304 -- abs was validated and stat'd as a regular file
	if err != nil {
		return fmt.Errorf("read workflow: %w", err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, abs, src, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse workflow: %w", err)
	}
	if f.Name.Name != "main" {
		return fmt.Errorf("workflow must use package main: got %q", f.Name.Name)
	}
	if err := validateRunAST(f); err != nil {
		return err
	}

	i := NewInterpreter()
	if _, err := i.Eval(string(src)); err != nil {
		return fmt.Errorf("workflow does not compile under Yaegi: %w", err)
	}

	if err := checkRunValue(i); err != nil {
		return err
	}
	return nil
}

func validateRunAST(f *ast.File) error {
	var runFn *ast.FuncDecl
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Run" || fn.Recv != nil {
			continue
		}
		if runFn != nil {
			return fmt.Errorf("multiple Run functions defined")
		}
		runFn = fn
	}
	if runFn == nil {
		return fmt.Errorf("workflow must define func Run")
	}
	if runFn.Type.Params.NumFields() != 3 {
		return fmt.Errorf("run must have 3 parameters (*kirarun.Context, *kirarun.Step, kirarun.Agents), got %d", runFn.Type.Params.NumFields())
	}
	want := []string{"*kirarun.Context", "*kirarun.Step", "kirarun.Agents"}
	for i, p := range runFn.Type.Params.List {
		got := typeExprString(p.Type)
		if got != want[i] {
			return fmt.Errorf("run parameter %d must be %s, got %s", i+1, want[i], got)
		}
	}
	if runFn.Type.Results.NumFields() != 1 {
		return fmt.Errorf("run must return exactly one value (error)")
	}
	res0 := runFn.Type.Results.List[0]
	if typeExprString(res0.Type) != "error" {
		return fmt.Errorf("run must return error, got %s", typeExprString(res0.Type))
	}
	return nil
}

func typeExprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExprString(t.X)
	case *ast.SelectorExpr:
		return typeExprString(t.X) + "." + t.Sel.Name
	case *ast.ParenExpr:
		return typeExprString(t.X)
	default:
		return ""
	}
}

func checkRunValue(i *interp.Interpreter) error {
	exports := i.Symbols("main")
	if len(exports) == 0 {
		return fmt.Errorf("internal: no main symbols after compile")
	}
	var runV reflect.Value
	for _, syms := range exports {
		if v, ok := syms["Run"]; ok {
			runV = v
			break
		}
	}
	if !runV.IsValid() {
		return fmt.Errorf("internal: Run symbol not found after compile")
	}
	want := reflect.TypeOf(func(*kirarun.Context, *kirarun.Step, kirarun.Agents) error { return nil })
	got := runV.Type()
	if got != want {
		return fmt.Errorf("run has wrong reflect type: got %s, want %s", got.String(), want.String())
	}
	return nil
}

// NewInterpreter returns a Yaegi interpreter with stdlib (excluding unsafe/syscall in upstream extract) and kira/kirarun.
func NewInterpreter() *interp.Interpreter {
	i := interp.New(interp.Options{})
	_ = i.Use(stdlib.Symbols)
	_ = i.Use(KirarunExports())
	i.ImportUsed()
	return i
}
