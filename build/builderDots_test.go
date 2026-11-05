package build

import (
	"fmt"
	"html/template"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"text/template/parse"

	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/templates"
	"github.com/hjson/hjson-go/v4"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Builder Dots
 *
 * These tests walk an embedded Template's action pipelines and the HTML
 * they render, tracking which builder each template receives as its dot,
 * and fail when a template calls an accessor that builder does not have.
 * html/template only finds those at render time.
 ******************************************/

// withStepBuilders maps each with-* step to the builder type its sub-steps run on
var withStepBuilders = map[string]reflect.Type{
	"with-annotation": reflect.TypeFor[Model](),
	"with-folder":     reflect.TypeFor[Model](),
	"with-following":  reflect.TypeFor[Model](),
	"with-message":    reflect.TypeFor[Model](),
	"with-rule":       reflect.TypeFor[Model](),
}

// knownUserInboxProblems lists what this check reports for user-inbox that is not a defect
var knownUserInboxProblems = []string{

	// Every /@me/newsfeed route requires sign-in, and only the signed-out branch calls StreamID
	`action "btn-social-outline" > btn-social-outline: build.Inbox has no StreamID`,
}

// TestBuilderDots_UserInbox confirms that every user-inbox template calls only what its builder has
func TestBuilderDots_UserInbox(t *testing.T) {

	checker := loadDotChecker(t, reflect.TypeFor[Inbox](), "user-inbox", "base-social")
	require.NotEmpty(t, checker.actions)

	// A new line is a new defect.  A missing one was fixed, and leaves knownUserInboxProblems
	require.ElementsMatch(t, knownUserInboxProblems, checker.checkAll())
}

// TestBuilderDots_FindsBUG330 confirms that a browse modal redrawn from inside with-annotation is reported
func TestBuilderDots_FindsBUG330(t *testing.T) {

	checker := loadDotChecker(t, reflect.TypeFor[Inbox](), "user-inbox", "base-social")

	checker.actions["bug-330"] = []any{
		map[string]any{"do": "with-annotation", "steps": []any{
			map[string]any{"do": "view-html", "method": "post", "file": "browse-annotation-modal"},
		}},
	}

	// The redraw that BUG-330 moved out of with-annotation
	problems := strings.Join(checker.checkAction("bug-330"), "\n")
	require.Contains(t, problems, `action "bug-330" > browse-annotation-modal > action "browse-annotation" > browse-annotation`)
	require.Contains(t, problems, "reply-button: build.Model has no CanReplyTo")
}

// TestBuilderDots_UserInboxLinks confirms that every link into the inbox names an action the inbox has
func TestBuilderDots_UserInboxLinks(t *testing.T) {

	checker := loadDotChecker(t, reflect.TypeFor[Inbox](), "user-inbox", "base-social")
	links := loadNewsfeedLinks(t, "user-inbox", "base-social")
	require.NotEmpty(t, links)

	require.Empty(t, danglingLinks(links, checker.actions))
}

// TestBuilderDots_Links pins which links are found, and which of them are reported
func TestBuilderDots_Links(t *testing.T) {

	t.Run("a literal link names its action", func(t *testing.T) {
		require.Equal(t, []string{"browse"}, findNewsfeedLinks(`<span hx-get="/@me/newsfeed/browse?url={{$url | urlquery}}">`))
	})

	t.Run("a BasePath link names its action", func(t *testing.T) {
		require.Equal(t, []string{"like-button"}, findNewsfeedLinks(`hx-post="{{.BasePath}}/like-button?url={{$url}}"`))
	})

	t.Run("a link inside an HTML comment is ignored", func(t *testing.T) {
		require.Empty(t, findNewsfeedLinks("<!--\n<a hx-get=\"/@me/newsfeed/gone\">\n-->"))
	})

	t.Run("a link inside an hjson comment is ignored", func(t *testing.T) {
		require.Empty(t, findNewsfeedLinks("\t\t// see /@me/newsfeed/gone\n\t\t# or /@me/newsfeed/gone"))
	})

	t.Run("a link built from a variable is not checked", func(t *testing.T) {
		require.Empty(t, findNewsfeedLinks(`hx-get="/@me/newsfeed/{{$action}}?url=x"`))
	})

	t.Run("the bare inbox URL names no action", func(t *testing.T) {
		require.Empty(t, findNewsfeedLinks(`<a href="/@me/newsfeed">Inbox</a>`))
	})

	t.Run("every link is found, in order", func(t *testing.T) {
		require.Equal(t, []string{"list", "browse", "list"}, findNewsfeedLinks(`/@me/newsfeed/list /@me/newsfeed/browse?x /@me/newsfeed/list`))
	})

	t.Run("only a link to a missing action is reported", func(t *testing.T) {
		links := map[string][]string{
			"browse":            {"user-inbox/list.html"},
			"replies-recursive": {"user-inbox/b.html", "user-inbox/a.html"},
		}
		require.Equal(t, []string{
			`user-inbox/a.html links "replies-recursive", which is not an action`,
			`user-inbox/b.html links "replies-recursive", which is not an action`,
		}, danglingLinks(links, map[string]any{"browse": nil}))
	})

	t.Run("nothing missing reports nothing", func(t *testing.T) {
		require.Empty(t, danglingLinks(nil, nil))
	})
}

// TestBuilderDots_Walker pins how the walker tracks a builder through a template
func TestBuilderDots_Walker(t *testing.T) {

	inbox := reflect.TypeFor[Inbox]()
	model := reflect.TypeFor[Model]()

	// check renders one template, from one action, with the given builder, and returns what it reports
	check := func(root reflect.Type, html map[string]string, actions map[string]any) []string {
		t.Helper()
		checker := newDotChecker(t, root, html, actions)
		return checker.checkAll()
	}

	view := func(file string) map[string]any {
		return map[string]any{"index": []any{map[string]any{"do": "view-html", "file": file}}}
	}

	t.Run("an accessor on the root dot", func(t *testing.T) {
		html := map[string]string{"page": `{{.CanReplyTo "x"}}`}
		require.Empty(t, check(inbox, html, view("page")))
		require.Equal(t, []string{`action "index" > page: build.Model has no CanReplyTo`}, check(model, html, view("page")))
	})

	t.Run("an accessor used as an argument", func(t *testing.T) {
		html := map[string]string{"page": `{{if eq "x" .NoSuchThing}}{{end}}`}
		require.Equal(t, []string{`action "index" > page: build.Inbox has no NoSuchThing`}, check(inbox, html, view("page")))
	})

	t.Run("a builder passed through array and index", func(t *testing.T) {
		html := map[string]string{
			"page":    `{{template "partial" (array "url" .)}}`,
			"partial": `{{- $builder := index . 1 -}}{{- $url := index . 0 -}}{{$builder.CanReplyTo $url}}{{$url.NotABuilder}}`,
		}
		require.Empty(t, check(inbox, html, view("page")))
		require.Equal(t, []string{`action "index" > page > partial: build.Model has no CanReplyTo`}, check(model, html, view("page")))
	})

	t.Run("range and with replace the dot, but not $", func(t *testing.T) {
		html := map[string]string{"page": `{{range .Rules}}{{.Anything}}{{$.CanReplyTo "x"}}{{end}}{{with .Message}}{{.Anything}}{{end}}`}
		require.Empty(t, check(inbox, html, view("page")))
		require.Equal(t, []string{
			`action "index" > page: build.Model has no Rules`,
			`action "index" > page: build.Model has no CanReplyTo`,
			`action "index" > page: build.Model has no Message`,
		}, check(model, html, view("page")))
	})

	t.Run("a variable declared inside a block stays there", func(t *testing.T) {
		html := map[string]string{"page": `{{$x := "text"}}{{if true}}{{$x := .}}{{$x.CanReplyTo "y"}}{{end}}{{$x.Anything}}`}
		require.Equal(t, []string{`action "index" > page: build.Model has no CanReplyTo`}, check(model, html, view("page")))
	})

	t.Run("a template that recurses into itself", func(t *testing.T) {
		html := map[string]string{"page": `{{.CanReplyTo "x"}}{{if false}}{{template "page" .}}{{end}}`}
		require.Len(t, check(model, html, view("page")), 1)
	})

	t.Run("View follows into the named action", func(t *testing.T) {
		html := map[string]string{"page": `{{.View "other"}}`, "other": `{{.CanReplyTo "x"}}`}
		actions := view("page")
		actions["other"] = []any{map[string]any{"do": "view-html"}}
		require.Empty(t, check(inbox, html, actions))
		require.Equal(t, []string{
			`action "index" > page > action "other" > other: build.Model has no CanReplyTo`,
			`action "other" > other: build.Model has no CanReplyTo`,
		}, check(model, html, actions))
	})

	t.Run("a with step hands its sub-steps a Model", func(t *testing.T) {
		html := map[string]string{"page": `{{.CanReplyTo "x"}}`}
		actions := map[string]any{"index": []any{
			map[string]any{"do": "with-annotation", "steps": []any{map[string]any{"do": "view-html", "file": "page"}}},
		}}
		require.Equal(t, []string{`action "index" > page: build.Model has no CanReplyTo`}, check(inbox, html, actions))
	})

	t.Run("steps nested in an if keep the builder", func(t *testing.T) {
		html := map[string]string{"page": `{{.CanReplyTo "x"}}`}
		actions := map[string]any{"index": []any{
			map[string]any{"do": "if", "condition": "{{.CanReplyTo `x`}}", "then": []any{map[string]any{"do": "view-html", "file": "page"}}},
		}}
		require.Empty(t, check(inbox, html, actions))
		require.Len(t, check(model, html, actions), 2)
	})

	t.Run("a step argument is checked against its builder", func(t *testing.T) {
		actions := map[string]any{"index": []any{
			map[string]any{"do": "set-query-param", "value": "{{.NoSuchThing}}", "options": []any{"a:{{.Nothing}}", "plain"}},
		}}
		require.Equal(t, []string{
			`action "index" > step "set-query-param": build.Inbox has no Nothing`,
			`action "index" > step "set-query-param": build.Inbox has no NoSuchThing`,
		}, check(inbox, nil, actions))
	})

	t.Run("an unknown with step, template, or action is reported", func(t *testing.T) {
		actions := map[string]any{"index": []any{
			map[string]any{"do": "with-something-new", "steps": []any{}},
			map[string]any{"do": "view-html", "file": "missing"},
		}}
		html := map[string]string{"page": `{{.View "nowhere"}}`}
		actions["other"] = []any{map[string]any{"do": "view-html", "file": "page"}}
		require.Equal(t, []string{
			`action "index" > step "with-something-new": add it to withStepBuilders`,
			`action "index" > missing: no such template`,
			`action "other" > page > action "nowhere": no such action`,
		}, check(inbox, html, actions))
	})
}

/******************************************
 * Checker
 ******************************************/

// dotValue is what the walker knows about a template value: a builder, an array holding builders, or neither
type dotValue struct {
	builder  reflect.Type
	elements []reflect.Type
}

// key identifies a dotValue, so that each template is walked once per kind of dot
func (value dotValue) key() string {
	return fmt.Sprint(value.builder, value.elements)
}

// dotChecker walks one Template's actions and HTML, collecting every accessor a builder lacks
type dotChecker struct {
	root     reflect.Type
	actions  map[string]any
	set      *template.Template
	funcs    template.FuncMap
	visited  map[string]bool
	reported map[string]bool
	problems []string
}

// loadDotChecker reads a Template folder and the folders it extends, child first, the way Template.Inherit merges them
func loadDotChecker(t *testing.T, root reflect.Type, folders ...string) *dotChecker {

	t.Helper()

	html := make(map[string]string)
	actions := make(map[string]any)

	for _, folder := range folders {

		// Read the actions, letting the child's copy win
		definition, err := os.ReadFile(filepath.Join("..", "_embed", "templates", folder, "template.hjson")) // #nosec G304 -- a checked-in template folder named by the test
		require.NoError(t, err)

		var parsed map[string]any
		require.NoErrorf(t, hjson.Unmarshal(definition, &parsed), "parsing %s/template.hjson", folder)

		folderActions, _ := parsed["actions"].(map[string]any)
		for actionID, action := range folderActions {
			actionMap, isMap := action.(map[string]any)
			require.Truef(t, isMap, "%s action %q is not an object", folder, actionID)
			if _, exists := actions[actionID]; !exists {
				actions[actionID] = actionMap["steps"]
			}
		}

		// Read the HTML, letting the child's copy win
		matches, err := filepath.Glob(filepath.Join("..", "_embed", "templates", folder, "*.html"))
		require.NoError(t, err)

		for _, path := range matches {
			name := strings.TrimSuffix(filepath.Base(path), ".html")
			if _, exists := html[name]; exists {
				continue
			}
			content, err := os.ReadFile(path) // #nosec G304 -- a file globbed from a checked-in template folder
			require.NoError(t, err)
			html[name] = string(content)
		}
	}

	return newDotChecker(t, root, html, actions)
}

// newDotChecker parses the given HTML templates, and readies a walk of the given actions with the root builder type
func newDotChecker(t *testing.T, root reflect.Type, html map[string]string, actions map[string]any) *dotChecker {

	t.Helper()

	funcs := templates.FuncMap(service.NewIcons())
	set := template.New("").Funcs(funcs)

	for name, content := range html {
		_, err := set.New(name).Parse(content)
		require.NoErrorf(t, err, "parsing %s", name)
	}

	return &dotChecker{root: root, actions: actions, set: set, funcs: funcs}
}

// checkAll walks every action with the root builder, and returns what it found, in a stable order
func (checker *dotChecker) checkAll() []string {

	actionIDs := make([]string, 0, len(checker.actions))
	for actionID := range checker.actions {
		actionIDs = append(actionIDs, actionID)
	}
	slices.Sort(actionIDs)

	result := make([]string, 0)
	for _, actionID := range actionIDs {
		result = append(result, checker.checkAction(actionID)...)
	}

	return result
}

// checkAction walks one action with the root builder, and returns what it found
func (checker *dotChecker) checkAction(actionID string) []string {
	checker.visited = make(map[string]bool)
	checker.reported = make(map[string]bool)
	checker.problems = nil
	checker.walkAction(actionID, checker.root, nil)
	return checker.problems
}

// report records one problem, prefixed by the path that reached it, once
func (checker *dotChecker) report(path []string, format string, args ...any) {

	if problem := strings.Join(path, " > ") + ": " + fmt.Sprintf(format, args...); !checker.reported[problem] {
		checker.reported[problem] = true
		checker.problems = append(checker.problems, problem)
	}
}

// walkAction walks the steps of the named action, run with the given builder
func (checker *dotChecker) walkAction(actionID string, builder reflect.Type, path []string) {

	path = append(slices.Clone(path), fmt.Sprintf("action %q", actionID))

	steps, exists := checker.actions[actionID]

	if !exists {
		checker.report(path, "no such action")
		return
	}

	// RULE: each action is walked once per builder, which also ends a View that calls itself
	key := "action|" + actionID + "|" + builder.String()
	if checker.visited[key] {
		return
	}
	checker.visited[key] = true

	checker.walkSteps(actionID, steps, builder, path)
}

// walkSteps walks a list of steps, following sub-steps, step arguments, and the HTML each view-html renders
func (checker *dotChecker) walkSteps(actionID string, steps any, builder reflect.Type, path []string) {

	list, _ := steps.([]any)

	for _, item := range list {

		step, _ := item.(map[string]any)
		do, _ := step["do"].(string)
		stepPath := append(slices.Clone(path), fmt.Sprintf("step %q", do))

		// Step arguments are templates rendered against this step's builder
		checker.walkStepArguments(step, builder, stepPath)

		// A with-* step hands its own sub-steps a different builder
		childBuilder := builder
		if strings.HasPrefix(do, "with-") {
			child, known := withStepBuilders[do]
			if !known {
				checker.report(stepPath, "add it to withStepBuilders")
				continue
			}
			childBuilder = child
		}

		// Walk every list of sub-steps, whatever the step calls it (steps, then, else, ...)
		for _, value := range step {
			if isStepList(value) {
				checker.walkSteps(actionID, value, childBuilder, path)
			}
		}

		// view-html renders its file, or the action's own name, with this builder
		if do == "view-html" {
			file, _ := step["file"].(string)
			if file == "" {
				file = actionID
			}
			checker.walkTemplate(file, dotValue{builder: builder}, path)
		}
	}
}

// walkStepArguments walks each top-level string argument of a step, or string in a list, that holds a template
func (checker *dotChecker) walkStepArguments(step map[string]any, builder reflect.Type, path []string) {

	keys := make([]string, 0, len(step))
	for key := range step {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, key := range keys {

		values := []any{step[key]}
		if list, isList := step[key].([]any); isList {
			values = list
		}

		for _, value := range values {

			text, isString := value.(string)
			if !isString || !strings.Contains(text, "{{") {
				continue
			}

			parsed, err := template.New("").Funcs(checker.funcs).Parse(text)
			if err != nil {
				checker.report(path, "cannot parse %q: %s", text, err)
				continue
			}

			dot := dotValue{builder: builder}
			checker.walkList(parsed.Tree.Root, dot, map[string]dotValue{"$": dot}, path)
		}
	}
}

// isStepList returns TRUE if the value is a list of steps
func isStepList(value any) bool {

	list, isList := value.([]any)
	if !isList || len(list) == 0 {
		return false
	}

	step, isMap := list[0].(map[string]any)
	if !isMap {
		return false
	}

	_, hasDo := step["do"]
	return hasDo
}

// walkTemplate walks the named HTML template, rendered with the given dot
func (checker *dotChecker) walkTemplate(name string, dot dotValue, path []string) {

	path = append(slices.Clone(path), name)

	tmpl := checker.set.Lookup(name)

	if tmpl == nil || tmpl.Tree == nil {
		checker.report(path, "no such template")
		return
	}

	// RULE: each template is walked once per kind of dot, which also ends a template that calls itself
	key := "template|" + name + "|" + dot.key()
	if checker.visited[key] {
		return
	}
	checker.visited[key] = true

	checker.walkList(tmpl.Tree.Root, dot, map[string]dotValue{"$": dot}, path)
}

// cloneScope copies a scope of template variables, so that a block's declarations stay inside it
func cloneScope(vars map[string]dotValue) map[string]dotValue {
	scope := make(map[string]dotValue, len(vars)+1)
	maps.Copy(scope, vars)
	return scope
}

// walkList walks each node in a list, in a scope of its own
func (checker *dotChecker) walkList(list *parse.ListNode, dot dotValue, vars map[string]dotValue, path []string) {

	if list == nil {
		return
	}

	scope := cloneScope(vars)

	for _, node := range list.Nodes {
		checker.walkNode(node, dot, scope, path)
	}
}

// walkNode walks one node of a template
func (checker *dotChecker) walkNode(node parse.Node, dot dotValue, vars map[string]dotValue, path []string) {

	switch typed := node.(type) {

	case *parse.ActionNode:
		checker.walkPipe(typed.Pipe, dot, vars, path)

	case *parse.IfNode:
		scope := cloneScope(vars)
		checker.walkPipe(typed.Pipe, dot, scope, path)
		checker.walkList(typed.List, dot, scope, path)
		checker.walkList(typed.ElseList, dot, scope, path)

	case *parse.RangeNode:
		scope := cloneScope(vars)
		checker.walkPipe(typed.Pipe, dot, scope, path)

		// A range binds its elements, which are never builders
		for _, variable := range typed.Pipe.Decl {
			scope[variable.Ident[0]] = dotValue{}
		}

		checker.walkList(typed.List, dotValue{}, scope, path)
		checker.walkList(typed.ElseList, dot, vars, path)

	case *parse.WithNode:
		scope := cloneScope(vars)
		value := checker.walkPipe(typed.Pipe, dot, scope, path)
		checker.walkList(typed.List, value, scope, path)
		checker.walkList(typed.ElseList, dot, vars, path)

	case *parse.TemplateNode:
		argument := dotValue{}
		if typed.Pipe != nil {
			argument = checker.walkPipe(typed.Pipe, dot, vars, path)
		}
		checker.walkTemplate(typed.Name, argument, path)
	}
}

// walkPipe checks every command in a pipe, binds its declared variables, and returns what the pipe yields
func (checker *dotChecker) walkPipe(pipe *parse.PipeNode, dot dotValue, vars map[string]dotValue, path []string) dotValue {

	for _, command := range pipe.Cmds {
		checker.walkCommand(command, dot, vars, path)
	}

	// Only a single, unpiped command can yield a builder
	result := dotValue{}
	if len(pipe.Cmds) == 1 {
		result = checker.evaluate(pipe.Cmds[0], dot, vars)
	}

	for _, variable := range pipe.Decl {
		vars[variable.Ident[0]] = result
	}

	return result
}

// walkCommand checks every accessor a command calls on a builder, and follows View into its action
func (checker *dotChecker) walkCommand(command *parse.CommandNode, dot dotValue, vars map[string]dotValue, path []string) {

	for index, argument := range command.Args {

		var builder reflect.Type
		var accessor string

		switch typed := argument.(type) {

		case *parse.FieldNode:
			builder, accessor = dot.builder, typed.Ident[0]

		case *parse.VariableNode:
			if len(typed.Ident) > 1 {
				builder, accessor = vars[typed.Ident[0]].builder, typed.Ident[1]
			}

		case *parse.PipeNode:
			checker.walkPipe(typed, dot, vars, path)
		}

		if builder == nil {
			continue
		}

		if !hasAccessor(builder, accessor) {
			checker.report(path, "%s has no %s", builder, accessor)
			continue
		}

		// RULE: .View "action" renders that action with a new builder of the same type
		if accessor == "View" && index == 0 && len(command.Args) > 1 {
			if actionID, isString := command.Args[1].(*parse.StringNode); isString {
				checker.walkAction(actionID.Text, builder, path)
			}
		}
	}
}

// evaluate returns what a single command yields: a builder, an array of values, or neither
func (checker *dotChecker) evaluate(command *parse.CommandNode, dot dotValue, vars map[string]dotValue) dotValue {

	operand := func(node parse.Node) dotValue {
		switch typed := node.(type) {
		case *parse.DotNode:
			return dot
		case *parse.VariableNode:
			if len(typed.Ident) == 1 {
				return vars[typed.Ident[0]]
			}
		case *parse.PipeNode:
			if len(typed.Cmds) == 1 {
				return checker.evaluate(typed.Cmds[0], dot, vars)
			}
		}
		return dotValue{}
	}

	first := command.Args[0]

	identifier, isIdentifier := first.(*parse.IdentifierNode)

	if !isIdentifier {
		if len(command.Args) == 1 {
			return operand(first)
		}
		return dotValue{}
	}

	switch identifier.Ident {

	case "array":
		result := dotValue{elements: make([]reflect.Type, 0, len(command.Args)-1)}
		for _, argument := range command.Args[1:] {
			result.elements = append(result.elements, operand(argument).builder)
		}
		return result

	case "index":
		if len(command.Args) != 3 {
			return dotValue{}
		}
		if number, isNumber := command.Args[2].(*parse.NumberNode); isNumber && number.IsInt && number.Int64 >= 0 {
			if container := operand(command.Args[1]); int(number.Int64) < len(container.elements) {
				return dotValue{builder: container.elements[number.Int64]}
			}
		}
	}

	return dotValue{}
}

// hasAccessor returns TRUE if a template can call the named method, or read the named field, on the type
func hasAccessor(builder reflect.Type, name string) bool {

	if _, exists := builder.MethodByName(name); exists {
		return true
	}

	if builder.Kind() == reflect.Struct {
		field, exists := builder.FieldByName(name)
		return exists && field.IsExported()
	}

	return false
}

/******************************************
 * Links
 ******************************************/

// newsfeedLink matches a link to an inbox action, written out or through .BasePath
var newsfeedLink = regexp.MustCompile(`(?:/@me/newsfeed|\{\{\s*\.BasePath\s*\}\})/([a-z][a-z0-9-]*)`)

// htmlComment matches an HTML comment, which the minifier deletes before a template is parsed
var htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)

// hjsonComment matches a whole-line hjson comment
var hjsonComment = regexp.MustCompile(`(?m)^\s*(?://|#).*$`)

// findNewsfeedLinks returns the action named by each inbox link in content, skipping comments
func findNewsfeedLinks(content string) []string {

	content = htmlComment.ReplaceAllString(content, "")
	content = hjsonComment.ReplaceAllString(content, "")
	result := make([]string, 0)

	for _, match := range newsfeedLink.FindAllStringSubmatch(content, -1) {
		result = append(result, match[1])
	}

	return result
}

// loadNewsfeedLinks reads each Template folder's definition, HTML, and hyperscript, child first,
// and maps each linked action to the files that link it
func loadNewsfeedLinks(t *testing.T, folders ...string) map[string][]string {

	t.Helper()

	result := make(map[string][]string)
	seen := make(map[string]bool)

	for _, folder := range folders {

		// Find the files, the way the Template loader does
		directory := filepath.Join("..", "_embed", "templates", folder)
		html, err := filepath.Glob(filepath.Join(directory, "*.html"))
		require.NoError(t, err)

		hyperscript, err := filepath.Glob(filepath.Join(directory, "hyperscript", "*._hs"))
		require.NoError(t, err)

		paths := append([]string{filepath.Join(directory, "template.hjson")}, html...)
		paths = append(paths, hyperscript...)

		// Collect each file's links, letting the child's copy of an HTML or hyperscript file win
		for _, path := range paths {

			filename := filepath.Base(path)
			location := folder + "/" + filename

			if (filename != "template.hjson") && seen[filename] {
				continue
			}
			seen[filename] = true

			content, err := os.ReadFile(path) // #nosec G304 -- a file globbed from a checked-in template folder
			require.NoError(t, err)

			for _, actionID := range findNewsfeedLinks(string(content)) {
				if !slices.Contains(result[actionID], location) {
					result[actionID] = append(result[actionID], location)
				}
			}
		}
	}

	return result
}

// danglingLinks reports each file that links an action missing from actions, in a stable order
func danglingLinks(links map[string][]string, actions map[string]any) []string {

	result := make([]string, 0)

	for actionID, files := range links {

		if _, exists := actions[actionID]; exists {
			continue
		}

		for _, file := range files {
			result = append(result, fmt.Sprintf("%s links %q, which is not an action", file, actionID))
		}
	}

	slices.Sort(result)
	return result
}
