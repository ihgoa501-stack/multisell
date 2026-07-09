// Docgen — auto-generates module reference Markdown from Go source code.
//
// Reads domain module routes.go, model.go and httpx/router.go to produce
// accurate, always-in-sync documentation of every:
//   - API endpoint (method + full path + handler)
//   - Model struct (field-level with type, JSON tag, GORM table)
//   - Permission/middleware group per module
//
// Usage:
//   cd backend-go && go run cmd/docgen/main.go          # generate to docs/auto/modules/
//   make docs                                           # same
//   make check-docs                                     # verify fresh (CI)

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

// --- Types ---

type Route struct {
	Method   string // GET POST PUT DELETE PATCH
	SubPath  string // as-registered in routes.go
	FullPath string // resolved with parent group prefix
	Handler  string // e.g. h.ListProducts
}

type ModelField struct {
	Name     string // Go field name
	Type     string // Go type, e.g. string, int64, decimal.Decimal
	JSONTag  string // from json:"..."
	GORMCol  string // from gorm:"column:...;..."
	IsID     bool
	NotNull  bool
	Default  string
	Tags     string // raw unparsed tag string
}

type ModelInfo struct {
	Name      string       // struct name
	TableName string       // from TableName() method or inferred
	Fields    []ModelField
	DBTable   string
}

type Module struct {
	Name       string   // e.g. "sku"
	Prefix     string   // base prefix from router.go
	Permission string   // permission group, e.g. "product.read"
	Routes     []Route
	Models     []ModelInfo
}

// --- Main ---

func main() {
	outDir := flag.String("out", "docs/auto/modules/", "output directory for per-module docs")
	flag.Parse()

	repoRoot := findRepoRoot()

	// 1. Parse router.go for mount prefix + permission per module
	mounts := parseMounts(filepath.Join(repoRoot, "internal/httpx/router.go"))
	log.Printf("Found %d module mounts", len(mounts))

	// 2. Walk all domain modules
	domainDir := filepath.Join(repoRoot, "internal/domain")
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
		m := parseModule(e.Name(), routesFile, modelFile, mounts)
		if len(m.Routes) > 0 || len(m.Models) > 0 {
			modules = append(modules, m)
		}
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Name < modules[j].Name })

	// 3. Write output
	absOut := *outDir
	if !filepath.IsAbs(absOut) {
		absOut = filepath.Join(repoRoot, absOut)
	}
	os.MkdirAll(absOut, 0755)

	for _, m := range modules {
		writeModuleDoc(filepath.Join(absOut, m.Name+".md"), m)
	}
	writeCatalogDoc(filepath.Join(absOut, "_route-catalog.md"), modules)
	writeIndexDoc(filepath.Join(absOut, "_index.md"), modules)

	log.Printf("Generated %d module docs + catalog in %s", len(modules), absOut)
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

// --- Router mount parsing ---

// Mount holds how a module is mounted in router.go.
type Mount struct {
	Prefix     string // e.g. "/api/v1"
	Permission string // e.g. "product.read" or ""
}

// parseMounts reads router.go and extracts per-module mount info.
func parseMounts(routerPath string) map[string]Mount {
	result := map[string]Mount{}
	data, err := os.ReadFile(routerPath)
	if err != nil {
		log.Printf("warn: cannot read %s: %v", routerPath, err)
		return result
	}
	content := string(data)

	// base prefix
	basePrefix := "/api/v1"
	if m := regexp.MustCompile(`api\s*:=\s*r\.Group\("([^"]+)"\)`).FindStringSubmatch(content); len(m) > 0 {
		basePrefix = m[1]
	}

	// Track group vars → permission
	type groupInfo struct {
		variable   string
		prefix     string
		permission string
	}
	var groups []groupInfo

	// protected := api.Group("") — base group, no extra permission
	groups = append(groups, groupInfo{variable: "protected", prefix: basePrefix, permission: ""})

	// Match group variables and their permission middleware
	// Format: productRoutes := protected.Group("", middleware.RequirePermission(db, "product.read"))
	grpRe := regexp.MustCompile(`(\w+)\s*:=\s*\w+\.Group\(`)
	permRe := regexp.MustCompile(`RequirePermission\([^,]+,\s*"([^"]+)"`)
	for _, loc := range grpRe.FindAllStringSubmatchIndex(content, -1) {
		matchStart := loc[0]
		varName := content[loc[2]:loc[3]]

		lineStart := strings.LastIndex(content[:matchStart], "\n")
		if lineStart < 0 { lineStart = 0 }
		lineEnd := strings.Index(content[lineStart+1:], "\n")
		if lineEnd < 0 { lineEnd = len(content) - lineStart }
		line := content[lineStart : lineStart+1+lineEnd]

		g := groupInfo{
			variable:   varName,
			prefix:     basePrefix,
			permission: "",
		}
		// Extract sub-path
		rest := content[matchStart:]
		if sp := regexp.MustCompile(`Group\("([^"]*)"`).FindStringSubmatch(rest); len(sp) > 0 && sp[1] != "" {
			g.prefix = basePrefix + sp[1]
		}
		// Extract permission middleware from the same line
		if pm := permRe.FindStringSubmatch(line); len(pm) > 0 {
			g.permission = pm[1]
		}
		groups = append(groups, g)
	}

	// Also match simpler: rbacRoutes := protected.Group("") (no permission middleware)
	simpleRe := regexp.MustCompile(`(\w+)\s*:=\s*(protected|\w+)\.Group\(`)
	for _, loc := range simpleRe.FindAllStringSubmatchIndex(content, -1) {
		matchStart := loc[0]
		varName := content[loc[2]:loc[3]]
		already := false
		for _, g := range groups {
			if g.variable == varName {
				already = true
				break
			}
		}
		if !already {
			// Extract sub-path from this occurrence
			rest := content[matchStart:]
			subPfx := ""
			if sp := regexp.MustCompile(`Group\("([^"]*)"`).FindStringSubmatch(rest); len(sp) > 0 && sp[1] != "" {
				subPfx = sp[1]
			}
			groups = append(groups, groupInfo{variable: varName, prefix: basePrefix + subPfx, permission: ""})
		}
	}

	// Match module.RegisterRoutes calls → extract module name + parent group
	regRe := regexp.MustCompile(`(\w+)\.RegisterRoutes\((\w+)`)
	for _, m := range regRe.FindAllStringSubmatch(content, -1) {
		modName := m[1]
		parentVar := m[2]
		for _, g := range groups {
			if g.variable == parentVar {
				result[modName] = Mount{Prefix: g.prefix, Permission: g.permission}
				break
			}
		}
	}
	return result
}

// --- Module parsing ---

func parseModule(name, routesFile, modelFile string, mounts map[string]Mount) Module {
	m := Module{
		Name:   name,
		Prefix: mounts[name].Prefix,
	}
	m.Permission = mounts[name].Permission

	fset := token.NewFileSet()

	// Parse routes
	if f, err := parser.ParseFile(fset, routesFile, nil, parser.ParseComments); err == nil {
		// Group variable → full prefix
		groupPfx := map[string]string{"rg": m.Prefix}
		// Find var := receiver.Group("subpath") assignments
		ast.Inspect(f, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok || as.Tok != token.DEFINE || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
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

		// Extract route registrations
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

			receiver := ""
			if id, ok := sel.X.(*ast.Ident); ok {
				receiver = id.Name
			}
			fullPath := strings.ReplaceAll(groupPfx[receiver]+subPath, "//", "/")

			m.Routes = append(m.Routes, Route{
				Method:   method,
				SubPath:  subPath,
				FullPath: fullPath,
				Handler:  handler,
			})
			return true
		})
	}

	// Parse model types with field details
	if f, err := parser.ParseFile(fset, modelFile, nil, parser.ParseComments); err == nil {
		// First pass: find TableName() methods
		tableNames := map[string]string{}
		ast.Inspect(f, func(n ast.Node) bool {
			fd, ok := n.(*ast.FuncDecl)
			if !ok || fd.Name.Name != "TableName" || fd.Recv == nil || len(fd.Recv.List) != 1 {
				return true
			}
			recvType := fd.Recv.List[0].Type
			typeName := ""
			if star, ok := recvType.(*ast.StarExpr); ok {
				if id, ok := star.X.(*ast.Ident); ok {
					typeName = id.Name
				}
			} else if id, ok := recvType.(*ast.Ident); ok {
				typeName = id.Name
			}
			if typeName == "" {
				return true
			}
			// Extract the string literal return value
			if len(fd.Body.List) > 0 {
				if ret, ok := fd.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) > 0 {
					if lit, ok := ret.Results[0].(*ast.BasicLit); ok {
						tableNames[typeName] = strings.Trim(lit.Value, "\"")
					}
				}
			}
			return true
		})

		// Second pass: struct types
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					continue
				}
				mi := ModelInfo{Name: ts.Name.Name, DBTable: tableNames[ts.Name.Name]}
				for _, field := range st.Fields.List {
					if len(field.Names) == 0 {
						continue // embedded
					}
					fieldName := field.Names[0].Name
					if !field.Names[0].IsExported() {
						continue
					}
					goType := exprString(field.Type)
					mf := ModelField{
						Name: fieldName,
						Type: goType,
					}
					if field.Tag != nil {
						raw := field.Tag.Value
						mf.Tags = raw
						// Extract JSON tag
						if jm := jsonTagRe.FindStringSubmatch(raw); len(jm) > 0 {
							mf.JSONTag = jm[1]
						}
						// Extract GORM column
						if cm := gormColRe.FindStringSubmatch(raw); len(cm) > 0 {
							mf.GORMCol = cm[1]
						}
						// Check constraints
						if gormNotNullRe.MatchString(raw) {
							mf.NotNull = true
						}
						if dm := gormDefaultRe.FindStringSubmatch(raw); len(dm) > 0 {
							mf.Default = dm[1]
						}
						if gormPKRe.MatchString(raw) {
							mf.IsID = true
						}
					}
					mi.Fields = append(mi.Fields, mf)
				}
				m.Models = append(m.Models, mi)
			}
		}
	}

	return m
}

var (
	jsonTagRe      = regexp.MustCompile(`json:"([^"]*)"`)
	gormColRe      = regexp.MustCompile(`column:([^;"]+)`)
	gormNotNullRe  = regexp.MustCompile(`not null`)
	gormDefaultRe  = regexp.MustCompile(`default:([^;"]+)`)
	gormPKRe       = regexp.MustCompile(`primaryKey`)
)

// exprString converts an AST expression to a Go type string.
func exprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprString(t.Elt)
		}
		return fmt.Sprintf("[%s]%s", exprString(t.Len), exprString(t.Elt))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprString(t.Key), exprString(t.Value))
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return fmt.Sprintf("%T", e)
	}
}

// --- Output writers ---

func writeModuleDoc(path string, m Module) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Module: `%s`\n\n", m.Name)
	fmt.Fprintf(f, "Package: `backend-go/internal/domain/%s/`\n\n", m.Name)

	if m.Prefix != "" {
		fmt.Fprintf(f, "**Base mount prefix:** `%s`\n", m.Prefix)
	}
	if m.Permission != "" {
		fmt.Fprintf(f, "**Required permission:** `%s`\n", m.Permission)
	}
	fmt.Fprintln(f)

	// --- Routes ---
	if len(m.Routes) > 0 {
		fmt.Fprintf(f, "## API Routes\n\n")
		fmt.Fprintf(f, "| Method | Path | Handler |\n")
		fmt.Fprintf(f, "|--------|------|--------|\n")
		sort.Slice(m.Routes, func(i, j int) bool {
			if m.Routes[i].FullPath != m.Routes[j].FullPath {
				return m.Routes[i].FullPath < m.Routes[j].FullPath
			}
			return m.Routes[i].Method < m.Routes[j].Method
		})
		for _, r := range m.Routes {
			fmt.Fprintf(f, "| `%s` | `%s` | `%s` |\n", r.Method, r.FullPath, r.Handler)
		}
		fmt.Fprintln(f)
	}

	// --- Models ---
	if len(m.Models) > 0 {
		fmt.Fprintf(f, "## Models\n\n")
		for _, mi := range m.Models {
			table := mi.DBTable
			if table == "" {
				table = "—"
			}
			fmt.Fprintf(f, "### `%s`\n", mi.Name)
			fmt.Fprintf(f, "**DB table:** `%s`\n\n", table)
			fmt.Fprintf(f, "| Field | Type | JSON | Column | Constraints |\n")
			fmt.Fprintf(f, "|-------|------|------|--------|-------------|\n")
			for _, mf := range mi.Fields {
				var constraints []string
				if mf.IsID {
					constraints = append(constraints, "PK")
				}
				if mf.NotNull {
					constraints = append(constraints, "NOT NULL")
				}
				if mf.Default != "" {
					constraints = append(constraints, "default:"+mf.Default)
				}
				col := mf.GORMCol
				if col == "" {
					col = "—"
				}
				fmt.Fprintf(f, "| `%s` | `%s` | `%s` | `%s` | %s |\n",
					mf.Name, mf.Type, mf.JSONTag, col, strings.Join(constraints, ", "))
			}
			fmt.Fprintln(f)
		}
	}

	fmt.Fprintln(f, "---")
	fmt.Fprintln(f, "_Auto-generated by `docgen`. Do not edit manually._")
}

// writeCatalogDoc generates a comprehensive route catalog (mirrors reference-module-catalog.md).
func writeCatalogDoc(path string, modules []Module) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Route Catalog (auto-generated)\n\n")
	fmt.Fprintf(f, "> Generated from Go source. Source of truth for all registered API routes.\n")
	fmt.Fprintf(f, "> Total modules: %d | Total routes: %d\n\n",
		len(modules), countRoutes(modules))

	// Sort modules by route count descending for overview
	sortedMods := append([]Module{}, modules...)
	sort.Slice(sortedMods, func(i, j int) bool {
		return len(sortedMods[i].Routes) > len(sortedMods[j].Routes)
	})

	fmt.Fprintf(f, "## Overview\n\n")
	fmt.Fprintf(f, "| Module | Routes | Permission | Package |\n")
	fmt.Fprintf(f, "|--------|--------|------------|--------|\n")
	for _, m := range sortedMods {
		perm := m.Permission
		if perm == "" {
			perm = "—"
		}
		fmt.Fprintf(f, "| [`%s`](%s.md) | %d | `%s` | `domain/%s/` |\n",
			m.Name, m.Name, len(m.Routes), perm, m.Name)
	}
	fmt.Fprintln(f)

	// Per-module route tables
	for _, m := range modules {
		if len(m.Routes) == 0 {
			continue
		}
		fmt.Fprintf(f, "## %s\n\n", m.Name)
		if m.Permission != "" {
			fmt.Fprintf(f, "**Permission:** `%s`  \n", m.Permission)
		}
		fmt.Fprintf(f, "**Prefix:** `%s`\n\n", m.Prefix)
		fmt.Fprintf(f, "| Method | Path | Handler |\n")
		fmt.Fprintf(f, "|--------|------|--------|\n")
		sort.Slice(m.Routes, func(i, j int) bool {
			if m.Routes[i].FullPath != m.Routes[j].FullPath {
				return m.Routes[i].FullPath < m.Routes[j].FullPath
			}
			return m.Routes[i].Method < m.Routes[j].Method
		})
		for _, r := range m.Routes {
			fmt.Fprintf(f, "| `%s` | `%s` | `%s` |\n", r.Method, r.FullPath, r.Handler)
		}
		fmt.Fprintln(f)
	}
}

func writeIndexDoc(path string, modules []Module) {
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Auto-generated Module Reference\n\n")
	fmt.Fprintf(f, "> Generated from Go source code. Update docs by running:\n")
	fmt.Fprintf(f, "> ```\n")
	fmt.Fprintf(f, "> cd backend-go && go run cmd/docgen/main.go\n")
	fmt.Fprintf(f, "> ```\n\n")
	fmt.Fprintf(f, "Regenerate with `make docs` in `backend-go/`.\n\n")

	fmt.Fprintf(f, "| Module | Routes | Models | Permission |\n")
	fmt.Fprintf(f, "|--------|--------|--------|------------|\n")
	for _, m := range modules {
		perm := m.Permission
		if perm == "" {
			perm = "—"
		}
		fmt.Fprintf(f, "| [%s](%s.md) | %d | %d | `%s` |\n",
			m.Name, m.Name, len(m.Routes), len(m.Models), perm)
	}
}

func countRoutes(modules []Module) int {
	n := 0
	for _, m := range modules {
		n += len(m.Routes)
	}
	return n
}
