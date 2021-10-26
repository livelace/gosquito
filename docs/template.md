### Template:

gosquito uses standard [golang template engine](https://pkg.go.dev/text/template). 

### Basic workflow:

1. Various plugins support templates, plugin parameters marked with "+" in "Template" column.
2. Template contains mix of text and [data item fields](concept.md).<br>Examples:  
```gotemplate
body = """
    <div align="right"><b>{{ .FLOW }}&nbsp;&nbsp;&nbsp;{{ .TIMEFORMAT }}</b></div>
    {{ .RSS.TITLE }}<br>
    {{ if .RSS.DESCRIPTION }}{{ .RSS.DESCRIPTION }}<br>{{end}}
    {{ if .RSS.CONTENT }}{{ .RSS.CONTENT }}<br><br>{{else}}<br>{{end}}
    {{ if .RSS.LINK }}{{ .RSS.LINK }}{{end}}
    """

body: '{"text": "{{ .ITER.VALUE | ToEscape }}"}'
```

### Additional template functions:

| Function      | Description                                |
| :------------ | :----------------------------------------- |
| FromBase64    | Decode Base64 string.                      |
| ToBase64      | Encode string to Base64.                   |
| ToEscape      | Escape string.                             |
| ToLower       | Change string characters to lower case.    |
| ToUpper       | Change string characters to upper case.    |


