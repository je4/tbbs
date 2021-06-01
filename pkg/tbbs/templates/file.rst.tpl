{{.Name}}
{{repeat "=" (len .Name)}}

Checksums
---------

{{range $cstype, $cs := .Checksums}}
{{$cstype}}::
{{range $line := (multiline $cs 32)}}
   {{$line}}{{end}}

{{end}}

Fileinfo
--------

{{if gt (len .Indexer.NSRL) 0}}
NSRL
^^^^
{{range $nsrl := .Indexer.NSRL}}
{{range $key, $val := $nsrl.File}}| File.{{$key}}: {{$val}}
{{end}}{{range $key, $val := $nsrl.FileMfG}}| FileMfg.{{$key}}: {{$val}}
{{end}}{{range $key, $val := $nsrl.OS}}| OS.{{$key}}: {{$val}}
{{end}}{{range $key, $val := $nsrl.OSMfg}}| OSMfg.{{$key}}: {{$val}}
{{end}}{{range $key, $val := $nsrl.Prod}}| Prod.{{$key}}: {{$val}}
{{end}}{{range $key, $val := $nsrl.ProdMfg}}| ProdMfg.{{$key}}: {{$val}}
{{end}}{{end}}
{{end}}
{{if gt (len .Indexer.Siegfried) 0}}
Siegfried
^^^^^^^^^
{{range $idx := .Indexer.Siegfried}}
| Pronom ID: {{$idx.ID}}
| Name: {{$idx.Name}}
| Mimetype: {{$idx.MIME}}
{{end}}
{{end}}
{{if not (eq .Indexer.FFProbe.Format.FormatName "")}}
FFProbe
^^^^^^^

Format
""""""
| {{.Indexer.FFProbe.Format.FormatName}} ({{.Indexer.FFProbe.Format.FormatLongName}})
| Duration: {{format_duration .Indexer.FFProbe.Format.Duration}}
| Bitrate: {{.Indexer.FFProbe.Format.BitRate}}

{{if gt (len .Indexer.FFProbe.Format.Tags) 0}}
Metadata
""""""""
{{range $key, $val := .Indexer.FFProbe.Format.Tags}}| {{$key}}: {{$val}}
{{end}}
{{end}}
{{end}}

#####
Files
#####