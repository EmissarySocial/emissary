/******************************************
 * Theme Template Contract Tests
 *
 * The pages in themeTemplateDots bypass the action
 * pipeline, so nothing but this file ties each template to
 * the dot its handler passes. html/template resolves the
 * accessors lazily, at render time, and a mismatch surfaces
 * as a 500 on a live page rather than as a build failure.
 *
 * The walk below follows `{{template "x" .}}` into the
 * shared partials, because that is where the contract
 * actually bites -- `{{.ThemeData "stylesheet"}}` in
 * "includes-head" is a hard error for any dot without that
 * exact method, since html/template accepts arguments only
 * for a real method. The same lookup without an argument
 * fails silently instead, which is why the problem went
 * unnoticed.
 ******************************************/

package handler

import (
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"text/template/parse"

	"github.com/EmissarySocial/emissary/build"
	"github.com/EmissarySocial/emissary/service"
	"github.com/EmissarySocial/emissary/tools/templates"
	"github.com/stretchr/testify/require"
)

// themeTemplateDots maps every Theme template that a handler renders directly to the type
// of the dot that handler passes it.  Add a row whenever a handler renders a Theme template.
var themeTemplateDots = map[string]any{
	"user-signin":         build.Theme{},
	"user-signout":        build.Theme{},
	"guest-signin":        build.Theme{},
	"reset-password":      build.Theme{},
	"reset-error":         build.Theme{},
	"reset-confirm":       build.Theme{},
	"checkout-claim":      build.Theme{},
	"reset-code":          build.PasswordReset{},
	"reset-code-invalid":  build.PasswordReset{},
	"reset-code-inactive": build.PasswordReset{},
	"oauth":               build.OAuthAuthorization{},
}

// TestThemeTemplateDots verifies that every accessor a handler-rendered Theme template
// calls on its dot exists on the type that handler passes, with a matching argument count.
func TestThemeTemplateDots(t *testing.T) {

	set := parseThemeTemplates(t)

	for templateName, dot := range themeTemplateDots {

		t.Run(templateName, func(t *testing.T) {

			root := set.Lookup(templateName)
			require.NotNil(t, root, "template %q is not defined in any theme", templateName)

			dotType := reflect.TypeOf(dot)

			for _, call := range rootDotCalls(t, set, root, make(map[string]bool)) {

				method, found := dotType.MethodByName(call.name)

				if !found {
					// A plain struct field is fine, but only when the template passes no
					// arguments -- html/template cannot invoke a field.
					if _, isField := dotType.FieldByName(call.name); isField && call.args == 0 {
						continue
					}

					t.Errorf("template %q calls .%s, which %s does not provide", templateName, call.name, dotType)
					continue
				}

				// MethodByName on a non-pointer type still counts the receiver in NumIn.
				wantArgs := method.Type.NumIn() - 1

				require.Equalf(t, wantArgs, call.args,
					"template %q calls .%s with %d argument(s), but %s.%s takes %d",
					templateName, call.name, call.args, dotType, call.name, wantArgs)
			}
		})
	}
}

// TestThemeTemplateDots_CoversEveryThemePage verifies that themeTemplateDots names every
// Theme template a handler can render.  Rendering "includes-head" is what identifies one:
// the pipeline reaches that partial only through "page", so any other template that pulls
// it in is served straight from a handler and needs its dot pinned above.
//
// Without this, a new theme page added without a table row would simply go unchecked --
// the very gap that let the original bug reach production.
func TestThemeTemplateDots_CoversEveryThemePage(t *testing.T) {

	const pipelinePage = "page"

	set := parseThemeTemplates(t)

	for _, tmpl := range set.Templates() {

		name := tmpl.Name()

		if name == pipelinePage || name == "" {
			continue
		}

		if !rendersPartial(set, tmpl, "includes-head", make(map[string]bool)) {
			continue
		}

		_, mapped := themeTemplateDots[name]
		require.Truef(t, mapped, "theme template %q renders includes-head but has no dot in themeTemplateDots", name)
	}

	// The reverse direction: a row naming a template that no longer exists is dead weight.
	for name := range themeTemplateDots {
		require.NotNilf(t, set.Lookup(name), "themeTemplateDots names %q, which no theme defines", name)
	}
}

// rendersPartial reports whether a template pulls in the named partial, directly or
// through another partial it renders.
func rendersPartial(set *template.Template, tmpl *template.Template, target string, visited map[string]bool) bool {

	if tmpl == nil || visited[tmpl.Name()] {
		return false
	}

	visited[tmpl.Name()] = true

	for _, name := range invokedPartials(tmpl) {

		if name == target {
			return true
		}

		if rendersPartial(set, set.Lookup(name), target, visited) {
			return true
		}
	}

	return false
}

// invokedPartials returns the name of every partial a template invokes, anywhere in its
// tree.  Unlike rootDotCalls, this descends into `range` and `with` bodies: they rebind the
// dot, but the partials inside them still render.
func invokedPartials(tmpl *template.Template) []string {

	if tmpl == nil || tmpl.Tree == nil {
		return nil
	}

	result := make([]string, 0)

	var walk func(node parse.Node)

	walk = func(node parse.Node) {

		switch typed := node.(type) {

		case *parse.ListNode:
			if typed == nil {
				return
			}
			for _, child := range typed.Nodes {
				walk(child)
			}

		case *parse.IfNode:
			walk(typed.List)
			walk(typed.ElseList)

		case *parse.RangeNode:
			walk(typed.List)
			walk(typed.ElseList)

		case *parse.WithNode:
			walk(typed.List)
			walk(typed.ElseList)

		case *parse.TemplateNode:
			result = append(result, typed.Name)
		}
	}

	walk(tmpl.Tree.Root)

	return result
}

// dotCall is one accessor invoked on a template's root dot, with the number of arguments
// the template supplies to it.
type dotCall struct {
	name string
	args int
}

// rootDotCalls collects every accessor invoked on the ROOT dot of a template, following
// `{{template "name" .}}` invocations into the shared partials.
func rootDotCalls(t *testing.T, set *template.Template, tmpl *template.Template, visited map[string]bool) []dotCall {

	t.Helper()

	if tmpl == nil || tmpl.Tree == nil || visited[tmpl.Name()] {
		return nil
	}

	visited[tmpl.Name()] = true

	result := make([]dotCall, 0)

	var walk func(node parse.Node)

	walk = func(node parse.Node) {

		switch typed := node.(type) {

		case *parse.ListNode:
			if typed == nil {
				return
			}
			for _, child := range typed.Nodes {
				walk(child)
			}

		case *parse.ActionNode:
			walk(typed.Pipe)

		case *parse.IfNode:
			walk(typed.Pipe)
			walk(typed.List)
			walk(typed.ElseList)

		case *parse.RangeNode, *parse.WithNode:
			// RULE: Skipped on purpose.  Both rebind the dot, so the accessors inside
			// them belong to the ranged element, not to the page's dot.

		case *parse.TemplateNode:
			// RULE: Only follow a partial handed the same dot.  `{{template "x" $y}}`
			// rebinds it, so its accessors belong to $y.
			if dotArg(typed.Pipe) {
				result = append(result, rootDotCalls(t, set, set.Lookup(typed.Name), visited)...)
			}

		case *parse.PipeNode:
			if typed == nil {
				return
			}
			for _, cmd := range typed.Cmds {
				walkCommand(cmd, &result)
			}
		}
	}

	walk(tmpl.Tree.Root)

	return result
}

// walkCommand records a root-dot accessor invoked by a single command, along with the
// number of arguments that follow it.  A FieldNode in any position other than the first is
// a bare value, so it takes no arguments.
func walkCommand(cmd *parse.CommandNode, result *[]dotCall) {

	if cmd == nil || len(cmd.Args) == 0 {
		return
	}

	for index, arg := range cmd.Args {

		switch typed := arg.(type) {

		case *parse.FieldNode:
			// Only the FIRST segment names an accessor on the dot; `.A.B` chains past it.
			if len(typed.Ident) == 0 {
				continue
			}

			args := 0
			if index == 0 {
				args = len(cmd.Args) - 1
			}

			*result = append(*result, dotCall{name: typed.Ident[0], args: args})

		case *parse.PipeNode:
			for _, nested := range typed.Cmds {
				walkCommand(nested, result)
			}
		}
	}
}

// dotArg reports whether a `{{template "name" X}}` invocation passes the current dot.
func dotArg(pipe *parse.PipeNode) bool {

	if pipe == nil {
		return false
	}

	for _, cmd := range pipe.Cmds {
		for _, arg := range cmd.Args {
			if _, isDot := arg.(*parse.DotNode); isDot {
				return true
			}
		}
	}

	return false
}

// parseThemeTemplates loads the embedded Theme templates the same way the Theme service
// does: theme-default's own files win, and theme-global fills in whatever is left.
func parseThemeTemplates(t *testing.T) *template.Template {

	t.Helper()

	set := template.New("").Funcs(templates.FuncMap(service.NewIcons()))

	// RULE: Child theme first.  "includes-head" is defined in BOTH themes, and the default
	// theme's copy is the one every inherited page actually renders.
	for _, theme := range []string{"theme-default", "theme-global"} {

		matches, err := filepath.Glob(filepath.Join("..", "_embed", "templates", theme, "*.html"))
		require.NoError(t, err)
		require.NotEmpty(t, matches, "no templates found for %s", theme)

		for _, path := range matches {

			name := strings.TrimSuffix(filepath.Base(path), ".html")

			// A template the child theme already defines is NOT overwritten by the parent.
			if set.Lookup(name) != nil {
				continue
			}

			content, err := os.ReadFile(path)
			require.NoError(t, err)

			_, err = set.New(name).Parse(string(content))
			require.NoErrorf(t, err, "parsing %s", path)
		}
	}

	return set
}
