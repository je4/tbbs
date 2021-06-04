Encryption Data for Archive

{{range $name, $keyiv := .KI}}
{{$name}}
{{repeat "=" (len $name)}}

{{$name}}-key:
{{range $line := (multiline $keyiv.Key 32)}}{{blocks $line " " 4}}
{{end}}
{{$name}}-iv:
{{range $line := (multiline $keyiv.IV 32)}}{{blocks $line " " 4}}
{{end}}
{{end}}