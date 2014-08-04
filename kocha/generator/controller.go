package generator

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/naoina/kocha"
	"github.com/naoina/kocha/util"
)

var (
	routeTableTypeName = reflect.TypeOf(kocha.RouteTable{}).Name()
)

// controllerGenerator is generator of controller.
type controllerGenerator struct {
	flag *flag.FlagSet
}

// Usage returns usage of controller generator.
func (g *controllerGenerator) Usage() string {
	return "controller NAME"
}

func (g *controllerGenerator) DefineFlags(fs *flag.FlagSet) {
	g.flag = fs
}

// Generate generate a controller templates.
func (g *controllerGenerator) Generate() {
	name := g.flag.Arg(0)
	if name == "" {
		util.PanicOnError(g, "abort: no NAME given")
	}
	camelCaseName := util.ToCamelCase(name)
	snakeCaseName := util.ToSnakeCase(name)
	receiverName := strings.ToLower(name)
	if len(receiverName) > 1 {
		receiverName = receiverName[:2]
	} else {
		receiverName = receiverName[:1]
	}
	data := map[string]interface{}{
		"Name":     camelCaseName,
		"Receiver": receiverName,
	}
	util.CopyTemplate(g,
		filepath.Join(SkeletonDir("controller"), "controller.go.template"),
		filepath.Join("app", "controller", snakeCaseName+".go"), data)
	util.CopyTemplate(g,
		filepath.Join(SkeletonDir("controller"), "view.html"),
		filepath.Join("app", "view", snakeCaseName+".html"), data)
	g.addRouteToFile(name)
}

func (g *controllerGenerator) addRouteToFile(name string) {
	routeFilePath := filepath.Join("config", "routes.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, routeFilePath, nil, 0)
	if err != nil {
		util.PanicOnError(g, "abort: failed to read file: %v", err)
	}
	routeStructName := util.ToCamelCase(name)
	routeName := util.ToSnakeCase(name)
	routeTableAST := findRouteTableAST(f)
	if routeTableAST == nil {
		return
	}
	routeASTs := findRouteASTs(routeTableAST)
	if routeASTs == nil {
		return
	}
	if isRouteDefined(routeASTs, routeStructName) {
		return
	}
	routeFile, err := os.OpenFile(routeFilePath, os.O_RDWR, 0644)
	if err != nil {
		util.PanicOnError(g, "abort: failed to open file: %v", err)
	}
	defer routeFile.Close()
	lastRouteAST := routeASTs[len(routeASTs)-1]
	offset := int64(fset.Position(lastRouteAST.End()).Offset)
	var buf bytes.Buffer
	if _, err := io.CopyN(&buf, routeFile, offset); err != nil {
		util.PanicOnError(g, "abort: failed to read file: %v", err)
	}
	buf.WriteString(fmt.Sprintf(`, {
	Name:       "%s",
	Path:       "/%s",
	Controller: &controller.%s{},
}`, routeName, routeName, routeStructName))
	if _, err := io.Copy(&buf, routeFile); err != nil {
		util.PanicOnError(g, "abort: failed to read file: %v", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		util.PanicOnError(g, "abort: failed to format file: %v", err)
	}
	if _, err := routeFile.WriteAt(formatted, 0); err != nil {
		util.PanicOnError(g, "abort: failed to update file: %v", err)
	}
}

var ErrRouteTableASTIsFound = errors.New("route table AST is found")

func findRouteTableAST(file *ast.File) (routeTableAST *ast.CompositeLit) {
	defer func() {
		if err := recover(); err != nil && err != ErrRouteTableASTIsFound {
			panic(err)
		}
	}()
	ast.Inspect(file, func(node ast.Node) bool {
		switch aType := node.(type) {
		case *ast.GenDecl:
			if aType.Tok != token.VAR {
				return false
			}
			ast.Inspect(aType, func(n ast.Node) bool {
				switch typ := n.(type) {
				case *ast.CompositeLit:
					switch t := typ.Type.(type) {
					case *ast.Ident:
						if t.Name == routeTableTypeName {
							routeTableAST = typ
							panic(ErrRouteTableASTIsFound)
						}
					}
				}
				return true
			})
		}
		return true
	})
	return routeTableAST
}

func findRouteASTs(clit *ast.CompositeLit) []*ast.CompositeLit {
	var routeASTs []*ast.CompositeLit
	for _, c := range clit.Elts {
		if a, ok := c.(*ast.CompositeLit); ok {
			routeASTs = append(routeASTs, a)
		}
	}
	return routeASTs
}

func isRouteDefined(routeASTs []*ast.CompositeLit, routeStructName string) bool {
	for _, a := range routeASTs {
		for _, elt := range a.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if kv.Key.(*ast.Ident).Name != "Controller" {
				continue
			}
			unary, ok := kv.Value.(*ast.UnaryExpr)
			if !ok {
				continue
			}
			lit, ok := unary.X.(*ast.CompositeLit)
			if !ok {
				continue
			}
			selector, ok := lit.Type.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			if selector.X.(*ast.Ident).Name == "controller" && selector.Sel.Name == routeStructName {
				return true
			}
		}
	}
	return false
}

func init() {
	Register("controller", &controllerGenerator{})
}
