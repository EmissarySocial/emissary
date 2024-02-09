# JSON Template

This library is an ugly hack that pretends to be a `template/json` package that I wish Go had, but doesn't.  When you provide it with a text template for a JSON document, this library can generate new documents for each set of value substitutions you provide.

This is a shockingly bad way to make sure that character sequences are escaped properly in the resulting JSON.  I wish there were another way, but I gots places to be, and I can't wait around right now for a better solution.  So if you have one, I can't wait to replace this with something better.

### Example Code

```go
// Define a "text-like" template containing JSON only.
template := New(`{"template": "Hello, {{.name}}!"}`)

// Pass in values and a result to populate
value := map[string]any{"name": "World"}
result := make(map[string]any)

// Execute it like a normal template.  The JSON is unmarshaled into the result.
if err := template.Execute(&result, value); err != nil {
	fmt.Println("Error: ", err)
}

fmt.Println(result["template"]) // Output: Hello, World!
```

### WTF Is Happening?

The short version is that this library wraps the `html/template` package, and passes your JSON templates through inside of a `<script>` tag wrapper that it discards before unmarshalling the finished results.

As I alluded to above, I'm disappointed in myself for thinking of this solution, just as I'm disappointed in your for considering using it. Perhaps someone out there will come up with a better way and we can rid ourselves of this nightmare once and for all.