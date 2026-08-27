package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/ast"
)

func runExplain(args []string) error {
	return runExplainWithOutput(args, os.Stdout)
}

func runExplainWithOutput(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(os.Stderr) // diagnostics go to stderr; out carries results only
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: gofhir-ecl explain <expression>")
		fmt.Fprintln(os.Stderr, "       gofhir-ecl explain -")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Parses the expression and prints the AST as an indented tree.")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("%w: expected exactly one expression argument", errUsage)
	}

	expr, err := readExpression(fs.Arg(0))
	if err != nil {
		return err
	}
	tree, err := ecl.Parse(expr)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	printAST(out, tree, 0)
	return nil
}

// printAST emits a one-line-per-node indented rendering of the AST. The
// per-type cases extract the meaningful identifier/value so the output is
// readable, not just the Go struct dump.
func printAST(out io.Writer, node ast.Expression, depth int) {
	indent := strings.Repeat("  ", depth)
	switch n := node.(type) {
	case *ast.ConceptRef:
		if n.Term != "" {
			fmt.Fprintf(out, "%sConceptRef %s |%s|\n", indent, n.ID, n.Term)
		} else {
			fmt.Fprintf(out, "%sConceptRef %s\n", indent, n.ID)
		}
	case *ast.Any:
		fmt.Fprintf(out, "%sAny (*)\n", indent)
	case *ast.Nested:
		fmt.Fprintf(out, "%sNested\n", indent)
		printAST(out, n.Inner, depth+1)
	case *ast.DescendantOf:
		printUnary(out, "DescendantOf <", n.Operand, indent, depth)
	case *ast.DescendantOrSelfOf:
		printUnary(out, "DescendantOrSelfOf <<", n.Operand, indent, depth)
	case *ast.ChildOf:
		printUnary(out, "ChildOf <!", n.Operand, indent, depth)
	case *ast.ChildOrSelfOf:
		printUnary(out, "ChildOrSelfOf <<!", n.Operand, indent, depth)
	case *ast.AncestorOf:
		printUnary(out, "AncestorOf >", n.Operand, indent, depth)
	case *ast.AncestorOrSelfOf:
		printUnary(out, "AncestorOrSelfOf >>", n.Operand, indent, depth)
	case *ast.ParentOf:
		printUnary(out, "ParentOf >!", n.Operand, indent, depth)
	case *ast.ParentOrSelfOf:
		printUnary(out, "ParentOrSelfOf >>!", n.Operand, indent, depth)
	case *ast.And:
		printBinary(out, "And", n.Left, n.Right, indent, depth)
	case *ast.Or:
		printBinary(out, "Or", n.Left, n.Right, indent, depth)
	case *ast.Minus:
		printBinary(out, "Minus", n.Left, n.Right, indent, depth)
	case *ast.MemberOf:
		if len(n.Fields) > 0 {
			fmt.Fprintf(out, "%sMemberOf ^[%s]\n", indent, strings.Join(n.Fields, ","))
		} else {
			fmt.Fprintf(out, "%sMemberOf ^\n", indent)
		}
		printAST(out, n.Operand, depth+1)
	case *ast.RefsetContainingAny:
		fmt.Fprintf(out, "%sRefsetContainingAny ^R\n", indent)
		printAST(out, n.Operand, depth+1)
	case *ast.DotExpression:
		fmt.Fprintf(out, "%sDotExpression .\n", indent)
		fmt.Fprintf(out, "%s  source:\n", indent)
		printAST(out, n.Source, depth+2)
		fmt.Fprintf(out, "%s  attribute:\n", indent)
		printAST(out, n.Attribute, depth+2)
	case *ast.Refined:
		fmt.Fprintf(out, "%sRefined :\n", indent)
		fmt.Fprintf(out, "%s  focus:\n", indent)
		printAST(out, n.Focus, depth+2)
		fmt.Fprintf(out, "%s  refinement: <%d groups, %d conjunction, %d disjunction>\n",
			indent, len(n.Refinement.Groups),
			len(n.Refinement.Conjunction), len(n.Refinement.Disjunction))
		if n.Refinement.AttrSet != nil {
			fmt.Fprintf(out, "%s  attributes:\n", indent)
			printAttrSet(out, n.Refinement.AttrSet, depth+2)
		}
	case *ast.Filtered:
		fmt.Fprintf(out, "%sFiltered (%d clauses)\n", indent, len(n.Filters))
		printAST(out, n.Operand, depth+1)
	case *ast.HistorySupplement:
		profile := n.Profile
		if profile == "" {
			profile = "HISTORY-MAX (default)"
		}
		fmt.Fprintf(out, "%sHistorySupplement %s\n", indent, profile)
		printAST(out, n.Operand, depth+1)
	case *ast.Top:
		printUnary(out, "Top !!>", n.Operand, indent, depth)
	case *ast.Bottom:
		printUnary(out, "Bottom !!<", n.Operand, indent, depth)
	case *ast.AltIdentifier:
		fmt.Fprintf(out, "%sAltIdentifier %s#%s\n", indent, n.Scheme, n.Code)
	default:
		fmt.Fprintf(out, "%s%T\n", indent, node)
	}
}

// printAttrSet renders the boolean tree of a refinement's attribute clauses.
// The operator matters: `a = x OR b = y` and `a = x, b = y` select different
// sets, and a flat listing cannot tell them apart.
func printAttrSet(out io.Writer, set *ast.AttributeSet, depth int) {
	if set == nil {
		return
	}
	indent := strings.Repeat("  ", depth)
	if set.Attr != nil {
		reverse := ""
		if set.Attr.Reverse {
			reverse = "R "
		}
		cardinality := ""
		if c := set.Attr.Cardinality; c != nil {
			maxVal := strconv.Itoa(c.Max)
			if c.Max == -1 {
				maxVal = "*"
			}
			cardinality = fmt.Sprintf("[%d..%s] ", c.Min, maxVal)
		}
		fmt.Fprintf(out, "%sAttribute %s%s%s\n", indent, cardinality, reverse, set.Attr.Op)
		fmt.Fprintf(out, "%s  name:\n", indent)
		printAST(out, set.Attr.Name, depth+2)
		fmt.Fprintf(out, "%s  value:\n", indent)
		printAST(out, set.Attr.Value, depth+2)
		return
	}
	fmt.Fprintf(out, "%s%s\n", indent, set.Op)
	for _, item := range set.Items {
		printAttrSet(out, item, depth+1)
	}
}

func printUnary(out io.Writer, label string, operand ast.Expression, indent string, depth int) {
	fmt.Fprintf(out, "%s%s\n", indent, label)
	printAST(out, operand, depth+1)
}

func printBinary(out io.Writer, label string, left, right ast.Expression, indent string, depth int) {
	fmt.Fprintf(out, "%s%s\n", indent, label)
	fmt.Fprintf(out, "%s  left:\n", indent)
	printAST(out, left, depth+2)
	fmt.Fprintf(out, "%s  right:\n", indent)
	printAST(out, right, depth+2)
}
