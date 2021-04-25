package service

import (
	"strings"
)

type Editor struct {
}

func NewEditor() *Editor {
	return &Editor{}
}

func (e Editor) Render(content string) string {
	html := `<div class="editor-DEFAULT">{{.}}</div>
	<script src="/static/ckeditor5/build/ckeditor.js"></script>
	<script>
	InlineEditor.create( document.querySelector( '.editor-DEFAULT' ), {
			
			toolbar: {
				items: [
					'heading',
					'|',
					'bold',
					'italic',
					'link',
					'bulletedList',
					'numberedList',
					'|',
					'outdent',
					'indent',
					'|',
					'imageUpload',
					'blockQuote',
					'insertTable',
					'mediaEmbed',
					'undo',
					'redo'
				]
			},
			language: 'en',
			image: {
				toolbar: [
					'imageTextAlternative',
					'imageStyle:full',
					'imageStyle:side'
				]
			},
			table: {
				contentToolbar: [
					'tableColumn',
					'tableRow',
					'mergeTableCells'
				]
			},
			licenseKey: '',
		} )
		.then( editor => {
			window.editor = editor;
		} )
		.catch( error => {
			console.error( 'Oops, something went wrong!' );
			console.error( 'Please, report the following error on https://github.com/ckeditor/ckeditor5/issues with the build id and the error stack trace:' );
			console.warn( 'Build id: 7tnd8anyafyl-nohdljl880ze' );
			console.error( error );
		} );
	</script>`

	return strings.Replace(html, "{{.}}", content, 1)
}
