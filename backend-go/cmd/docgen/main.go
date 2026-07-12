// Docgen — auto-generates module reference Markdown from Go source code.
//
// Reads domain module routes.go, model.go and httpx/router.go to produce
// accurate, always-in-sync documentation of every:
//   - API endpoint (method + full path + handler)
//   - Model struct (field-level with type, JSON tag, GORM table)
//   - Permission/middleware group per module
//   - EventBus subscriptions and scheduler tasks
//   - AI Agent roster
//   - Frontend page catalog (Next.js app router)
//
// Usage:
//   cd backend-go && go run cmd/docgen/main.go          # generate all
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
	"strconv"
	"strings"
)

// --- Types ---

type Route struct {
	Method   string
	SubPath  string
	FullPath string
	Handler  string
}

type ModelField struct {
	Name    string
	Type    string
	JSONTag string
	GORMCol string
	IsID    bool
	NotNull bool
	Default string
}

type ModelInfo struct {
	Name      string
	TableName string
	Fields    []ModelField
}

type Module struct {
	Name       string
	Prefix     string
	Permission string
	Routes     []Route
	Models     []ModelInfo
}

type SchedTask struct {
	ID            string
	AgentID       string
	DecisionPoint string
	Interval      string
	Description   string
}

type Subscription struct {
	Topic   string
	Handler string // short description of what it does
	AgentID string // agent ID if it's an agent execution
}

type AgentDef struct {
	ID             string
	Name           string
	Squad          string
	DecisionPoints []string
	Description    string
	Autonomy       string
}

type FrontendPage struct {
	Path    string // URL path
	Dir     string // file system path relative to (main)
	HasPage bool   // has page.tsx
	HasID   bool   // is dynamic route [id]
}

// --- Main ---

func main() {
	outDir := flag.String("out", "docs/auto/", "output directory")
	repoRoot := flag.String("repo", "", "repo root (auto-detected if empty)")
	flag.Parse()

	rr := *repoRoot
	if rr == "" {
		rr = findRepoRoot()
	}
	// If rr is backend-go/, the project root is one level up.
	// frontend pages live at projectRoot/frontend-next/src/...
	projectRoot := rr
	if _, err := os.Stat(filepath.Join(rr, "go.mod")); err == nil && filepath.Base(rr) == "backend-go" {
		projectRoot = filepath.Dir(rr)
	}

	// 1. Module routes + models
	mounts := parseMounts(filepath.Join(rr, "internal/httpx/router.go"))
	domainDir := filepath.Join(rr, "internal/domain")
	entries, _ := os.ReadDir(domainDir)
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

	// 2. Scheduler + EventBus
	tasks := parseSchedulerTasks(filepath.Join(rr, "internal/httpx/router.go"))
	subs := parseSubscriptions(filepath.Join(rr, "internal/httpx/router.go"))

	// 3. Agent roster
	agents := parseAgents(rr)

	// 4. Frontend pages
	pages := parseFrontendPages(filepath.Join(projectRoot, "frontend-next/src/app/(main)"))

	// Write output
	absOut := *outDir
	if !filepath.IsAbs(absOut) {
		absOut = filepath.Join(rr, absOut)
	}
	os.MkdirAll(absOut, 0755)

	for _, m := range modules {
		writeModuleDoc(filepath.Join(absOut, "modules", m.Name+".md"), m)
	}
	writeCatalogDoc(filepath.Join(absOut, "modules", "_route-catalog.md"), modules)
	writeIndexDoc(filepath.Join(absOut, "modules", "_index.md"), modules)

	writeFrontendPagesDoc(filepath.Join(absOut, "_frontend-pages.md"), pages)
	writeEventFlowDoc(filepath.Join(absOut, "_event-flow.md"), tasks, subs)
	writeAgentRosterDoc(filepath.Join(absOut, "_agent-roster.md"), agents)

	log.Printf("Generated %d module docs + 3 cross-cutting reports in %s", len(modules), absOut)
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

type Mount struct {
	Prefix     string
	Permission string
}

func parseMounts(routerPath string) map[string]Mount {
	result := map[string]Mount{}
	data, err := os.ReadFile(routerPath)
	if err != nil {
		return result
	}
	content := string(data)

	basePrefix := "/api/v1"
	if m := regexp.MustCompile(`api\s*:=\s*r\.Group\("([^"]+)"\)`).FindStringSubmatch(content); len(m) > 0 {
		basePrefix = m[1]
	}

	type groupInfo struct {
		variable   string
		prefix     string
		permission string
	}
	var groups []groupInfo
	groups = append(groups, groupInfo{variable: "protected", prefix: basePrefix, permission: ""})

	grpRe := regexp.MustCompile(`(\w+)\s*:=\s*\w+\.Group\(`)
	permRe := regexp.MustCompile(`RequirePermission\([^,]+,\s*"([^"]+)"`)
	for _, loc := range grpRe.FindAllStringSubmatchIndex(content, -1) {
		matchStart := loc[0]
		varName := content[loc[2]:loc[3]]
		lineEnd := findLineEnd(content, matchStart)
		line := content[findLineStart(content, matchStart) : matchStart+lineEnd]
		g := groupInfo{variable: varName, prefix: basePrefix, permission: ""}
		if sp := regexp.MustCompile(`Group\("([^"]*)"`).FindStringSubmatch(content[matchStart:]); len(sp) > 0 && sp[1] != "" {
			g.prefix = basePrefix + sp[1]
		}
		if pm := permRe.FindStringSubmatch(line); len(pm) > 0 {
			g.permission = pm[1]
		}
		groups = append(groups, g)
	}

	// simpleRe: catch any group variable not already covered
	for _, loc := range regexp.MustCompile(`(\w+)\s*:=\s*(protected|\w+)\.Group\(`).FindAllStringSubmatchIndex(content, -1) {
		varName := content[loc[2]:loc[3]]
		already := false
		for _, g := range groups {
			if g.variable == varName {
				already = true
				break
			}
		}
		if !already {
			subPfx := ""
			if sp := regexp.MustCompile(`Group\("([^"]*)"`).FindStringSubmatch(content[loc[0]:]); len(sp) > 0 && sp[1] != "" {
				subPfx = sp[1]
			}
			groups = append(groups, groupInfo{variable: varName, prefix: basePrefix + subPfx, permission: ""})
		}
	}

	regRe := regexp.MustCompile(`(\w+)\.RegisterRoutes\((\w+)`)
	for _, m := range regRe.FindAllStringSubmatch(content, -1) {
		for _, g := range groups {
			if g.variable == m[2] {
				result[m[1]] = Mount{Prefix: g.prefix, Permission: g.permission}
				break
			}
		}
	}
	return result
}

func findLineStart(s string, pos int) int {
	ls := strings.LastIndex(s[:pos], "\n")
	if ls < 0 {
		return 0
	}
	return ls + 1
}

func findLineEnd(s string, pos int) int {
	le := strings.Index(s[pos:], "\n")
	if le < 0 {
		return len(s) - pos
	}
	return le
}

// --- Module parsing ---

func parseModule(name, routesFile, modelFile string, mounts map[string]Mount) Module {
	m := Module{Name: name}
	if mt, ok := mounts[name]; ok {
		m.Prefix = mt.Prefix
		m.Permission = mt.Permission
	}

	fset := token.NewFileSet()
	groupPfx := map[string]string{"rg": m.Prefix}

	if f, err := parser.ParseFile(fset, routesFile, nil, parser.ParseComments); err == nil {
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
			groupPfx[as.Lhs[0].(*ast.Ident).Name] = parentPfx + subPfx
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
			m.Routes = append(m.Routes, Route{Method: method, SubPath: subPath, FullPath: fullPath, Handler: handler})
			return true
		})
	}

	// Parse model types with field details
	parseModelFields(fset, modelFile, &m)
	return m
}

func parseModelFields(fset *token.FileSet, modelFile string, m *Module) {
	f, err := parser.ParseFile(fset, modelFile, nil, parser.ParseComments)
	if err != nil {
		return
	}

	// TableName methods
	tableNames := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "TableName" || fd.Recv == nil || len(fd.Recv.List) != 1 {
			return true
		}
		t := fd.Recv.List[0].Type
		typeName := ""
		if star, ok := t.(*ast.StarExpr); ok {
			if id, ok := star.X.(*ast.Ident); ok {
				typeName = id.Name
			}
		} else if id, ok := t.(*ast.Ident); ok {
			typeName = id.Name
		}
		if typeName == "" || len(fd.Body.List) == 0 {
			return true
		}
		if ret, ok := fd.Body.List[0].(*ast.ReturnStmt); ok && len(ret.Results) > 0 {
			if lit, ok := ret.Results[0].(*ast.BasicLit); ok {
				tableNames[typeName] = strings.Trim(lit.Value, "\"")
			}
		}
		return true
	})

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
			mi := ModelInfo{Name: ts.Name.Name, TableName: tableNames[ts.Name.Name]}
			for _, field := range st.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				fn := field.Names[0]
				if !fn.IsExported() {
					continue
				}
				mf := ModelField{Name: fn.Name, Type: exprString(field.Type)}
				if field.Tag != nil {
					raw := field.Tag.Value
					if jm := jsonTagRe.FindStringSubmatch(raw); len(jm) > 0 {
						mf.JSONTag = jm[1]
					}
					if cm := gormColRe.FindStringSubmatch(raw); len(cm) > 0 {
						mf.GORMCol = cm[1]
					}
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

var (
	jsonTagRe     = regexp.MustCompile(`json:"([^"]*)"`)
	gormColRe     = regexp.MustCompile(`column:([^;"]+)`)
	gormNotNullRe = regexp.MustCompile(`not null`)
	gormDefaultRe = regexp.MustCompile(`default:([^;"]+)`)
	gormPKRe      = regexp.MustCompile(`primaryKey`)
)

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

// --- Scheduler task parsing ---

func parseSchedulerTasks(routerPath string) []SchedTask {
	var tasks []SchedTask
	data, err := os.ReadFile(routerPath)
	if err != nil {
		return tasks
	}
	content := string(data)

	// Match: sched.Register(scheduler.Task{...})
	blockRe := regexp.MustCompile(`(?s)scheduler\.Task\{(.*?)\}`)
	for _, b := range blockRe.FindAllStringSubmatch(content, -1) {
		block := b[1]
		id := extractStr(block, `ID:\s*"([^"]+)"`)
		agentID := extractStr(block, `AgentID:\s*"([^"]+)"`)
		dp := extractStr(block, `DecisionPoint:\s*"([^"]+)"`)
		interval := extractStr(block, `Interval:\s*([^,}]+)`)
		desc := extractStr(block, `Description:\s*"([^"]*)"`)
		if id == "" {
			continue
		}
		// Normalize interval
		interval = normalizeInterval(interval)
		tasks = append(tasks, SchedTask{
			ID: id, AgentID: agentID, DecisionPoint: dp,
			Interval: interval, Description: desc,
		})
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks
}

func normalizeInterval(s string) string {
	s = strings.TrimSpace(s)
	// time.Minute * 5 → 5m, time.Hour * 1 → 1h
	if strings.Contains(s, "time.Minute") {
		if n := extractInt(s); n > 0 {
			return fmt.Sprintf("%dm", n)
		}
	}
	if strings.Contains(s, "time.Hour") {
		if n := extractInt(s); n > 0 {
			return fmt.Sprintf("%dh", n)
		}
	}
	return s
}

func extractInt(s string) int {
	re := regexp.MustCompile(`\*?\s*(\d+)`)
	if m := re.FindStringSubmatch(s); len(m) > 0 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

func extractStr(s, pattern string) string {
	re := regexp.MustCompile(pattern)
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	return ""
}

// --- EventBus subscription parsing ---

func parseSubscriptions(routerPath string) []Subscription {
	var subs []Subscription
	data, err := os.ReadFile(routerPath)
	if err != nil {
		return subs
	}
	content := string(data)

	// Match: bus.Subscribe("topic", func...) — extract topic and a brief description
	re := regexp.MustCompile(`bus\.Subscribe\("([^"]+)"`)
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		topic := m[1]
		agentID := ""
		// Infer agent ID from topic pattern
		if strings.HasPrefix(topic, "scheduler.tick.") {
			agentID = strings.TrimPrefix(topic, "scheduler.tick.")
		}
		subs = append(subs, Subscription{Topic: topic, AgentID: agentID})
	}
	sort.Slice(subs, func(i, j int) bool { return subs[i].Topic < subs[j].Topic })
	return subs
}

// --- Agent roster parsing ---

func parseAgents(repoRoot string) []AgentDef {
	// Read agents.go for agent IDs
	implFile := filepath.Join(repoRoot, "internal/agent/impl/agents.go")
	implData, _ := os.ReadFile(implFile)
	idRe := regexp.MustCompile(`"([A-Z]\d+)"\s*:`)
	idMatches := idRe.FindAllStringSubmatch(string(implData), -1)

	var agents []AgentDef
	for _, m := range idMatches {
		agents = append(agents, AgentDef{ID: m[1]})
	}

	// Read registry.go for descriptions (line-based regex scan)
	regFile := filepath.Join(repoRoot, "internal/ai/registry.go")
	regData, _ := os.ReadFile(regFile)
	lines := strings.Split(string(regData), "\n")

	currentID := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := regexp.MustCompile(`ID:\s*"(\w+)"`).FindStringSubmatch(trimmed); len(m) > 0 {
			currentID = m[1]
		}
		if currentID == "" {
			continue
		}

		if m := regexp.MustCompile(`Name:\s*"([^"]+)"`).FindStringSubmatch(trimmed); len(m) > 0 {
			setAgentField(&agents, currentID, 1, m[1])
		}
		if m := regexp.MustCompile(`Squad:\s*"([^"]+)"`).FindStringSubmatch(trimmed); len(m) > 0 {
			setAgentField(&agents, currentID, 2, m[1])
		}
		if m := regexp.MustCompile(`Description:\s*"([^"]+)"`).FindStringSubmatch(trimmed); len(m) > 0 {
			setAgentField(&agents, currentID, 3, m[1])
		}
		if m := regexp.MustCompile(`Autonomy:\s*"([^"]+)"`).FindStringSubmatch(trimmed); len(m) > 0 {
			setAgentField(&agents, currentID, 4, m[1])
		}
		if strings.HasPrefix(trimmed, "DecisionPoints:") || strings.Contains(trimmed, "DecisionPoints: []string{") {
			var dps []string
			dpsRe := regexp.MustCompile(`"([^"]+)"`)
			for _, dm := range dpsRe.FindAllStringSubmatch(trimmed, -1) {
				dps = append(dps, dm[1])
			}
			// DecisionPoints values in string context include the field name
			// Filter out "DecisionPoints" key
			var filtered []string
			for _, d := range dps {
				if d != "DecisionPoints" && !strings.HasPrefix(d, "[]string{") {
					filtered = append(filtered, d)
				}
			}
			if len(filtered) > 0 {
				setAgentDps(&agents, currentID, filtered)
			}
		}
	}

	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })
	return agents
}

func setAgentField(agents *[]AgentDef, id string, field int, val string) {
	for i := range *agents {
		if (*agents)[i].ID == id {
			switch field {
			case 1:
				(*agents)[i].Name = val
			case 2:
				(*agents)[i].Squad = val
			case 3:
				(*agents)[i].Description = val
			case 4:
				(*agents)[i].Autonomy = val
			}
		}
	}
}

func setAgentDps(agents *[]AgentDef, id string, dps []string) {
	for i := range *agents {
		if (*agents)[i].ID == id {
			(*agents)[i].DecisionPoints = dps
		}
	}
}

// --- Frontend page parsing ---

func parseFrontendPages(mainDir string) []FrontendPage {
	var pages []FrontendPage
	entries, err := os.ReadDir(mainDir)
	if err != nil {
		return pages
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		pagePath := name
		hasPage := false
		hasID := strings.HasPrefix(name, "[") && strings.HasSuffix(name, "]")

		// Check for page.tsx
		if _, err := os.Stat(filepath.Join(mainDir, name, "page.tsx")); err == nil {
			hasPage = true
		} else {
			// Check nested directories
			subDir := filepath.Join(mainDir, name)
			subEntries, _ := os.ReadDir(subDir)
			for _, se := range subEntries {
				if se.IsDir() {
					subName := se.Name()
					if _, err := os.Stat(filepath.Join(subDir, subName, "page.tsx")); err == nil {
						hasPage = true
						subPagePath := name + "/" + subName
						subHasID := strings.HasPrefix(subName, "[")
						pages = append(pages, FrontendPage{Path: subPagePath, Dir: name + "/" + subName, HasPage: true, HasID: subHasID})
					}
				}
			}
		}
		pages = append(pages, FrontendPage{Path: pagePath, Dir: name, HasPage: hasPage, HasID: hasID})
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Path < pages[j].Path })
	return pages
}

// --- Output writers ---

func writeModuleDoc(path string, m Module) {
	os.MkdirAll(filepath.Dir(path), 0755)
	f, _ := os.Create(path)
	if f == nil {
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

	if len(m.Routes) > 0 {
		fmt.Fprintf(f, "## API Routes\n\n| Method | Path | Handler |\n|--------|------|--------|\n")
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

	if len(m.Models) > 0 {
		fmt.Fprintf(f, "## Models\n\n")
		for _, mi := range m.Models {
			table := mi.TableName
			if table == "" {
				table = "—"
			}
			fmt.Fprintf(f, "### `%s`\n**DB table:** `%s`\n\n| Field | Type | JSON | Column | Constraints |\n|-------|------|------|--------|-------------|\n", mi.Name, table)
			for _, mf := range mi.Fields {
				var cons []string
				if mf.IsID {
					cons = append(cons, "PK")
				}
				if mf.NotNull {
					cons = append(cons, "NOT NULL")
				}
				if mf.Default != "" {
					cons = append(cons, "default:"+mf.Default)
				}
				col := mf.GORMCol
				if col == "" {
					col = "—"
				}
				fmt.Fprintf(f, "| `%s` | `%s` | `%s` | `%s` | %s |\n", mf.Name, mf.Type, mf.JSONTag, col, strings.Join(cons, ", "))
			}
			fmt.Fprintln(f)
		}
	}

	fmt.Fprintln(f, "---\n_Auto-generated by `docgen`. Do not edit manually._")
}

func writeCatalogDoc(path string, modules []Module) {
	os.MkdirAll(filepath.Dir(path), 0755)
	f, _ := os.Create(path)
	if f == nil {
		return
	}
	defer f.Close()

	totalRoutes := 0
	for _, m := range modules {
		totalRoutes += len(m.Routes)
	}
	fmt.Fprintf(f, "# Route Catalog (auto-generated)\n\n> Total modules: %d | Total routes: %d\n\n", len(modules), totalRoutes)

	sortedMods := append([]Module{}, modules...)
	sort.Slice(sortedMods, func(i, j int) bool { return len(sortedMods[i].Routes) > len(sortedMods[j].Routes) })
	fmt.Fprintf(f, "## Overview\n\n| Module | Routes | Permission | Package |\n|--------|--------|------------|--------|\n")
	for _, m := range sortedMods {
		perm := m.Permission
		if perm == "" {
			perm = "—"
		}
		fmt.Fprintf(f, "| [`%s`](%s.md) | %d | `%s` | `domain/%s/` |\n", m.Name, m.Name, len(m.Routes), perm, m.Name)
	}
	fmt.Fprintln(f)

	routedModules := make([]Module, 0, len(modules))
	for _, m := range modules {
		if len(m.Routes) > 0 {
			routedModules = append(routedModules, m)
		}
	}
	for i, m := range routedModules {
		fmt.Fprintf(f, "## %s\n\n", m.Name)
		if m.Permission != "" {
			fmt.Fprintf(f, "**Permission:** `%s`\n\n", m.Permission)
		}
		fmt.Fprintf(f, "**Prefix:** `%s`\n\n| Method | Path | Handler |\n|--------|------|--------|\n", m.Prefix)
		sort.Slice(m.Routes, func(i, j int) bool {
			if m.Routes[i].FullPath != m.Routes[j].FullPath {
				return m.Routes[i].FullPath < m.Routes[j].FullPath
			}
			return m.Routes[i].Method < m.Routes[j].Method
		})
		for _, r := range m.Routes {
			fmt.Fprintf(f, "| `%s` | `%s` | `%s` |\n", r.Method, r.FullPath, r.Handler)
		}
		if i < len(routedModules)-1 {
			fmt.Fprintln(f)
		}
	}
}

func writeIndexDoc(path string, modules []Module) {
	os.MkdirAll(filepath.Dir(path), 0755)
	f, _ := os.Create(path)
	if f == nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Auto-generated Module Reference\n\n")
	fmt.Fprintf(f, "> Regenerate with `make docs` in `backend-go/`.\n\n")
	fmt.Fprintf(f, "| Module | Routes | Models | Permission |\n|--------|--------|--------|------------|\n")
	for _, m := range modules {
		perm := m.Permission
		if perm == "" {
			perm = "—"
		}
		fmt.Fprintf(f, "| [%s](%s.md) | %d | %d | `%s` |\n", m.Name, m.Name, len(m.Routes), len(m.Models), perm)
	}
}

func writeFrontendPagesDoc(path string, pages []FrontendPage) {
	os.MkdirAll(filepath.Dir(path), 0755)
	f, _ := os.Create(path)
	if f == nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Frontend Pages (auto-generated)\n\n")
	fmt.Fprintf(f, "> Generated from `frontend-next/src/app/(main)/` directory structure.\n\n")
	fmt.Fprintf(f, "| Route | Has page.tsx | Dynamic |\n|-------|-------------|---------|\n")
	for _, p := range pages {
		dynamic := ""
		if p.HasID {
			dynamic = "✅"
		}
		pageStatus := "✅"
		if !p.HasPage {
			pageStatus = "—"
		}
		fmt.Fprintf(f, "| `/%s` | %s | %s |\n", p.Path, pageStatus, dynamic)
	}
	fmt.Fprintln(f)

	fmt.Fprintln(f, "---\n_Auto-generated by `docgen`. Do not edit manually._")
}

func writeEventFlowDoc(path string, tasks []SchedTask, subs []Subscription) {
	os.MkdirAll(filepath.Dir(path), 0755)
	f, _ := os.Create(path)
	if f == nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Event Flow (auto-generated)\n\n")
	fmt.Fprintf(f, "> Generated from `internal/httpx/router.go`.\n\n")

	// Scheduler tasks
	if len(tasks) > 0 {
		fmt.Fprintf(f, "## Scheduler Tasks\n\n")
		fmt.Fprintf(f, "| Task ID | Agent | Decision Point | Interval | Description |\n")
		fmt.Fprintf(f, "|---------|-------|----------------|----------|-------------|\n")
		for _, t := range tasks {
			fmt.Fprintf(f, "| `%s` | `%s` | `%s` | %s | %s |\n", t.ID, t.AgentID, t.DecisionPoint, t.Interval, t.Description)
		}
		fmt.Fprintln(f)
	}

	// EventBus subscriptions
	if len(subs) > 0 {
		fmt.Fprintf(f, "## EventBus Subscriptions\n\n")
		fmt.Fprintf(f, "| Topic | Pattern | Description |\n")
		fmt.Fprintf(f, "|-------|---------|-------------|\n")
		for _, s := range subs {
			desc := ""
			if strings.HasPrefix(s.Topic, "scheduler.tick.") {
				desc = fmt.Sprintf("Scheduled trigger for `%s`", s.AgentID)
			} else if strings.HasPrefix(s.Topic, "agent.decided.") {
				desc = "Agent decision pipeline"
			} else if strings.HasPrefix(s.Topic, "supplychain.") {
				desc = "Supply chain event"
			}
			fmt.Fprintf(f, "| `%s` | %s | %s |\n", s.Topic, globStyle(s.Topic), desc)
		}
		fmt.Fprintln(f)
	}
	fmt.Fprintln(f, "---\n_Auto-generated by `docgen`. Do not edit manually._")
}

func globStyle(topic string) string {
	if strings.Contains(topic, "*") {
		return "glob"
	}
	return "exact"
}

func writeAgentRosterDoc(path string, agents []AgentDef) {
	os.MkdirAll(filepath.Dir(path), 0755)
	f, _ := os.Create(path)
	if f == nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "# Agent Roster (auto-generated)\n\n")
	fmt.Fprintf(f, "> Generated from `internal/agent/impl/agents.go` and `internal/ai/registry.go`.\n")
	fmt.Fprintf(f, "> Total: %d agents\n\n", len(agents))

	fmt.Fprintf(f, "| ID | Name | Squad | Decision Points | Autonomy |\n")
	fmt.Fprintf(f, "|----|------|-------|----------------|----------|\n")
	for _, a := range agents {
		dps := strings.Join(a.DecisionPoints, ", ")
		fmt.Fprintf(f, "| `%s` | %s | `%s` | `%s` | `%s` |\n", a.ID, a.Name, a.Squad, dps, a.Autonomy)
	}
	fmt.Fprintln(f)

	fmt.Fprintln(f, "---\n_Auto-generated by `docgen`. Do not edit manually._")
}
