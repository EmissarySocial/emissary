package tests

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// emulatedSource names production code that this suite reproduces instead of calling,
// whose fingerprint TestEmulatedSources checks
type emulatedSource struct {
	Pattern string   // A file, or a glob of files, relative to this package
	Names   []string // Function names, "Receiver.method" names, or package-level vars; "*" for all
	UsedBy  string   // The code in this package that reproduces it
}

var emulatedSources = []emulatedSource{
	{Pattern: "../build/step_EditModelObject.go", Names: []string{"StepEditModelObject.Post", "StepEditModelObject.getForm"}, UsedBy: "editModelObjectCase"},
	{Pattern: "../build/step_AddModelObject.go", Names: []string{"StepAddModelObject.Post"}, UsedBy: "addModelObjectCase"},
	{Pattern: "../build/step_SetData.go", Names: []string{"StepSetData.Get", "StepSetData.Post", "StepSetData.setURLPaths"}, UsedBy: "setDataUse.apply"},
	{Pattern: "../build/step_EditTable.go", Names: []string{"StepTableEditor.Post"}, UsedBy: "editTableCase"},
	{Pattern: "../build/step_ReadForm.go", Names: []string{"StepReadForm.Post"}, UsedBy: "readFormCase"},
	{Pattern: "../build/step_EditTemplate.go", Names: []string{"StepEditTemplate.Post", "StepEditTemplate.isTemplateAllowed", "StepEditTemplate.listTemplates"}, UsedBy: "editTemplateCase, allowedTemplates"},
	{Pattern: "../build/step_EditWidget.go", Names: []string{"StepEditWidget.Post"}, UsedBy: "editWidgetCase"},
	{Pattern: "../build/step_EditRegistration.go", Names: []string{"StepEditRegistration.Post"}, UsedBy: "editRegistrationCase"},
	{Pattern: "../build/step_With*.go", Names: []string{"*"}, UsedBy: "subContext"},
	{Pattern: "../build/builder_*.go", Names: []string{"*.schema", "*.object", "*.PropertyForm"}, UsedBy: "templateContexts, subContext, widgetContext"},
	{Pattern: "../build/*.go", Names: []string{"*.GetPointer", "*.GetStringOK", "*.GetBoolOK", "*.GetIntOK", "*.GetInt64OK", "*.GetFloatOK"}, UsedBy: "builderStandIn (set-data defaults)"},
	{Pattern: "../build/utilities.go", Names: []string{"executeTemplate"}, UsedBy: "renderLiteral"},
	{Pattern: "../build/widget_save.go", Names: []string{"*"}, UsedBy: "widgetContext"},
	{Pattern: "../handler/admin.go", Names: []string{"buildAdmin_GetBuilder"}, UsedBy: "templateContexts"},
	{Pattern: "../model/template.go", Names: []string{"templateModelRegistry", "templateModelForName", "Template.BaseSchema", "Template.NewObject"}, UsedBy: "templateContexts, isStreamModel"},
	{Pattern: "../model/step/step.go", Names: []string{"New"}, UsedBy: "nestedSteps, newPostCase, unpinnedPostReaders"},
	{Pattern: "../server/factory_core.go", Names: []string{"factoryCore.EditorJS"}, UsedBy: "loadGroup, editorJS"},
	{Pattern: "module:github.com/benpate/rosetta/schema/schema_set.go", Names: []string{"Schema.SetURLValues"}, UsedBy: "setURLValuesInOrder"},
}

// TestEmulatedSources fails when production code that this suite reproduces has changed.
// See AGENTS.md for what to do when it fails.
func TestEmulatedSources(t *testing.T) {

	actual := map[string]string{}

	for _, source := range emulatedSources {

		// A dependency's file is found in the module cache, but keyed by its module path
		if modulePath, ok := strings.CutPrefix(source.Pattern, "module:"); ok {
			for name, fingerprint := range fingerprints(t, moduleFile(t, modulePath), source.Names) {
				actual[source.Pattern+" "+name+" (used by "+source.UsedBy+")"] = fingerprint
			}
			continue
		}

		files, err := filepath.Glob(source.Pattern)
		require.NoError(t, err)
		require.NotEmpty(t, files, "no files match %s", source.Pattern)

		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			for name, fingerprint := range fingerprints(t, file, source.Names) {
				actual[filepath.ToSlash(file)+" "+name+" (used by "+source.UsedBy+")"] = fingerprint
			}
		}
	}

	filename := filepath.Join("testdata", "emulated-sources.golden.json")

	if *update {
		encoded, err := json.MarshalIndent(actual, "", "\t")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filename, append(encoded, '\n'), 0o600))
		return
	}

	encoded, err := os.ReadFile(filename) // #nosec G304 -- a fixed name in testdata
	require.NoError(t, err, "missing golden file; run go test ./tests -update")

	expected := map[string]string{}
	require.NoError(t, json.Unmarshal(encoded, &expected))

	for _, name := range sortedKeys(expected) {
		require.Contains(t, actual, name, "production code that this suite reproduces was removed or renamed")
		require.Equal(t, expected[name], actual[name], "production code changed; update the emulation it is used by, then run -update: %s", name)
	}

	require.Equal(t, sortedKeys(expected), sortedKeys(actual), "new production code matches an emulated pattern; review it, then run -update")
}

// moduleFile resolves "module/path/file.go" to the file in the module version this build uses
func moduleFile(t *testing.T, modulePath string) string {

	t.Helper()

	for module := modulePath; strings.Contains(module, "/"); module = module[:strings.LastIndex(module, "/")] {

		output, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", module).Output() // #nosec G204 -- module paths come from the fixed list above
		if err != nil {
			continue
		}

		if directory := strings.TrimSpace(string(output)); directory != "" {
			return filepath.Join(directory, strings.TrimPrefix(modulePath, module+"/"))
		}
	}

	require.Fail(t, "module not found", modulePath)
	return ""
}

// fingerprints hashes the named declarations in a Go file, ignoring comments and formatting
func fingerprints(t *testing.T, file string, names []string) map[string]string {

	t.Helper()

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, file, nil, parser.SkipObjectResolution)
	require.NoError(t, err)

	result := map[string]string{}

	for _, declaration := range parsed.Decls {

		name := declarationName(declaration)

		if name == "" || !matchesName(name, names) {
			continue
		}

		var buffer bytes.Buffer
		require.NoError(t, printer.Fprint(&buffer, fileSet, declaration))

		sum := sha256.Sum256(buffer.Bytes())
		result[name] = hex.EncodeToString(sum[:8])
	}

	return result
}

// declarationName returns "Function", "Receiver.method", or "variable" for a declaration
func declarationName(declaration ast.Decl) string {

	switch typed := declaration.(type) {

	case *ast.FuncDecl:
		if typed.Recv == nil || len(typed.Recv.List) == 0 {
			return typed.Name.Name
		}
		return receiverName(typed.Recv.List[0].Type) + "." + typed.Name.Name

	case *ast.GenDecl:
		if typed.Tok != token.VAR || len(typed.Specs) != 1 {
			return ""
		}
		if spec, ok := typed.Specs[0].(*ast.ValueSpec); ok && len(spec.Names) == 1 {
			return spec.Names[0].Name
		}
	}

	return ""
}

// receiverName returns the type name of a method receiver, without any pointer
func receiverName(expression ast.Expr) string {

	switch typed := expression.(type) {
	case *ast.StarExpr:
		return receiverName(typed.X)
	case *ast.Ident:
		return typed.Name
	case *ast.IndexExpr:
		return receiverName(typed.X)
	}

	return ""
}

// matchesName reports whether a declaration name matches any pattern: an exact name,
// "*" for everything, or "*.method" for that method on any receiver
func matchesName(name string, patterns []string) bool {

	for _, pattern := range patterns {

		if pattern == "*" || pattern == name {
			return true
		}

		if method, ok := strings.CutPrefix(pattern, "*."); ok && strings.HasSuffix(name, "."+method) {
			return true
		}
	}

	return false
}
