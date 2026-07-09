package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Route struct {
	Method   string
	Path     string
	Handler  string
	FullPath string
}

type Module struct {
	Name   string
	Prefix string
	Routes []Route
	Models []string
}

func main() {
	outDir := flag.String("out", "docs/auto/modules/", "output directory")
	flag.Parse()

	repoRoot := findRepoRoot()
	if repoRoot == "" {
		log.Fatal("cannot find repo root")
	}

	domainDir := filepath.Join(repoRoot, "internal/domain")
	parentPrefixes := parseRouterFile(filepath.Join(repoRoot, "internal/httpx/router.go"))
	log.Printf("Found %d module→prefix mappings", len(parentPrefixes))

	entries, err := os.ReadDir(domainDir)
	if err != nil {
		log.Fatal(err)
	}

	var modules []Module
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		routesFile := filepath.Join(domainDir, e.Name(), "routes.go")
		modelFile := filepath.Join(domainDir, e.Name(), "model.go")
		if _, err := os.Stat(routesFile); os.IsNotExist(err) {
			continue
		}
		m := parseModule(e.Name(), routesFile, modelFile, parentPrefixes)
		if len(m.Routes) > 0 {
			modules = append(modules, m)
		}
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Name < modules[j].Name })

	writeOutput(modules, *outDir, repoRoot)
	log.Printf("Generated %d module docs in %s", len(modules), *outDir)
}

func findRepoRoot() string {
	cwd, _ := os.Getwd()
	for d := cwd; d != "/"; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
	}
	return ""
}

func parseRouterFile(path string) map[string]string {
	result := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return result
	}
	content := string(data)

	basePrefix := "/api/v1"
	if m := regexp.MustCompile(`api\s*:=\s*r\.Group\("([^"]+)"\)`).FindStringSubmatch(content); len(m) > 0 {
		basePrefix = m[1]
	}

	// Track group variable assignments
	groupVars := map[string]string{
		"protected": basePrefix,
	}

	// e.g. productRoutes := protected.Group("")
	grp := regexp.MustCompile(`(\w+)\s*:=\s*protected\.Group\(`)
	for _, m := range grp.FindAllStringSubmatch(content, -1) {
		groupVars[m[1]] = basePrefix
	}

	// e.g. rbacRoutes := protected.Group(""
	other := regexp.MustCompile(`(\w+)\s*:=\s*\w+\.Group\(`)
	for _, m := range other.FindAllStringSubmatch(content, -1) {
		if _, ok := groupVars[m[1]]; !ok {
			groupVars[m[1]] = basePrefix
		}
	}

	// Map module registration -> parent prefix
	reg := regexp.MustCompile(`(\w+)\.RegisterRoutes\((\w+)`)
	for _, m := range reg.FindAllStringSubmatch(content, -1) {
		if p, ok := groupVars[m[2]]; ok {
			result[m[1]] = p
		}
	}
	return result
}

func parseModule(name, routesFile, modelFile string, parentPrefixes map[string]string) Module {
	m := Module{Name: name, Prefix: parentPrefixes[name]}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, routesFile, nil, parser.ParseComments)
	if err != nil {
		return m
	}

	// Map group variable names to their full prefix path.
	// Start with the function parameter "rg" which equals the mount prefix.
	groupPfx := map[string]string{"rg": m.Prefix}
	parseGroups(f, fset, groupPfx)

	// Walk all call expressions to find GET/POST/PUT/DELETE/PATCH
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		method := sel.Sel.Name
		switch method {
		case "GET", "POST", "PUT", "DELETE", "PATCH":
		default:
			return true
		}
		if len(call.Args) < 2 {
			return true
		}
		pathLit, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}
		subPath := strings.Trim(pathLit.Value, "\"")

		receiver := ""
		if id, ok := sel.X.(*ast.Ident); ok {
			receiver = id.Name
		}

		handler := ""
		switch h := call.Args[1].(type) {
		case *ast.Ident:
			handler = h.Name
		case *ast.SelectorExpr:
			if id, ok := h.X.(*ast.Ident); ok {
				handler = id.Name + "." + h.Sel.Name
			} else {
				handler = h.Sel.Name
			}
		}

		fullPath := groupPfx[receiver] + subPath
		fullPath = strings.ReplaceAll(fullPath, "//", "/")

		m.Routes = append(m.Routes, Route{
			Method:   method,
			Path:     subPath,
			Handler:  handler,
			FullPath: fullPath,
		})
		return true
	})

	// Parse model types
	if _, err := os.Stat(modelFile); err == nil {
		if ma, err := parser.ParseFile(fset, modelFile, nil, parser.ParseComments); err == nil {
			for _, decl := range ma.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || !ts.Name.IsExported() {
						continue
					}
					if st, ok := ts.Type.(*ast.StructType); ok && st.Fields != nil {
						m.Models = append(m.Models, ts.Name.Name)
					}
				}
			}
		}
	}
	return m
}

// parseGroups finds all var := receiver.Group("subpath") assignments and populates groupPfx.
func parseGroups(f *ast.File, fset *token.FileSet, groupPfx map[string]string) {
	ast.Inspect(f, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || as.Tok != token.DEFINE {
			return true
		}
		if len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		call, ok := as.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Group" {
			return true
		}
		receiver := ""
		if id, ok := sel.X.(*ast.Ident); ok {
			receiver = id.Name
		}
		parentPfx := groupPfx[receiver]
		if parentPfx == "" {
			return true
		}
		subPfx := ""
		if len(call.Args) > 0 {
			if lit, ok := call.Args[0].(*ast.BasicLit); ok {
				subPfx = strings.Trim(lit.Value, "\"")
			}
		}
		varName := as.Lhs[0].(*ast.Ident).Name
		groupPfx[varName] = parentPfx + subPfx
		return true
	})
}

func writeOutput(modules []Module, outDir, repoRoot string) {
	absOut := outDir
	if !filepath.IsAbs(absOut) {
		absOut = filepath.Join(repoRoot, absOut)
	}
	if err := os.MkdirAll(absOut, 0755); err != nil {
		log.Fatal(err)
	}

	for _, m := range modules {
		path := filepath.Join(absOut, m.Name+".md")
		f, err := os.Create(path)
		if err != nil {
			continue
		}

		fmt.Fprintf(f, "# Module: %s\n\n", m.Name)
		if m.Prefix != "" {
			fmt.Fprintf(f, "> Base prefix: `%s`\n\n", m.Prefix)
		}
		fmt.Fprintf(f, "Package: `backend-go/internal/domain/%s/`\n\n", m.Name)

		if len(m.Models) > 0 {
			fmt.Fprintf(f, "## Models\n\n")
			for _, mo := range m.Models {
				fmt.Fprintf(f, "- `%s`\n", mo)
			}
			fmt.Fprintln(f)
		}

		fmt.Fprintf(f, "## API Routes\n\n")
		fmt.Fprintf(f, "| Method | Path | Handler |\n")
		fmt.Fprintf(f, "|--------|------|--------|\n")
		for _, r := range m.Routes {
			fmt.Fprintf(f, "| %s | `%s` | `%s` |\n", pad(r.Method), r.FullPath, r.Handler)
		}
		fmt.Fprintln(f)

		fmt.Fprintln(f, "---")
		fmt.Fprintln(f, "_Auto-generated by docgen. Do not edit manually._")
		f.Close()
	}

	// Write index
	f, err := os.Create(filepath.Join(absOut, "_index.md"))
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Auto-generated Module Reference\n\n")
	fmt.Fprintf(f, "> Generated from Go source code. Update docs by running:\n")
	fmt.Fprintf(f, "> ```\n")
	fmt.Fprintf(f, "> cd backend-go && go run cmd/docgen/main.go\n")
	fmt.Fprintf(f, "> ```\n\n")
	fmt.Fprintf(f, "| Module | Routes | Models |\n")
	fmt.Fprintf(f, "|--------|--------|--------|\n")
	for _, m := range modules {
		fmt.Fprintf(f, "| [%s](%s.md) | %d | %d |\n", m.Name, m.Name, len(m.Routes), len(m.Models))
	}
}

func pad(s string) string { return fmt.Sprintf("%-6s", s) }
