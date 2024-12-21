package executor

import (
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

func RemoveGolangMain(sourceCode string) string {

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", sourceCode, parser.ParseComments)
	if err != nil {
		return sourceCode
	}

	var newDecls []ast.Decl
	for _, decl := range node.Decls {
		if fnDecl, ok := decl.(*ast.FuncDecl); ok {
			if fnDecl.Name.Name == "main" {
				// Skip the main() function
				continue
			}
		}
		newDecls = append(newDecls, decl)
	}

	node.Decls = newDecls

	var output strings.Builder
	err = formatNode(&output, fset, node)
	if err != nil {
		return sourceCode
	}

	return output.String()
}

func formatNode(builder *strings.Builder, fset *token.FileSet, node interface{}) error {
	return formatNodeIndent(builder, fset, node, "")
}

func formatNodeIndent(builder *strings.Builder, fset *token.FileSet, node interface{}, indent string) error {
	return printerConfig.Fprint(builder, fset, node)
}

var printerConfig = &printer.Config{
	Mode:     printer.UseSpaces | printer.TabIndent,
	Tabwidth: 8,
}
