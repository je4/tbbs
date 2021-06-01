*********
Überblick
*********

Inhalte
=======
.. tabularcolumns:: |l|l|l|r|

{{.Contents.DrawTable}}

Integritätstests
================

{{range $loc, $table := .Tests}}
{{repeat "-" (len (print "Speicher: " $loc))}}
Speicher: {{$loc}}
{{repeat "-" (len (print "Speicher: " $loc))}}
.. tabularcolumns:: |l|l|l|l|

{{$table.DrawTable}}
{{end}}

Transfer (File to Storage / Time)
=================================

{{range $loc, $transfer := .Transfer}}| Transfer to "{{$loc}}" from {{$transfer.Start}} to {{$transfer.End}}: {{$transfer.Status}}
{{end}}

SHA512 Prüfsummen Übersicht
===========================
{{range $file, $sha512 := .SHA512}}
{{$file}} [SHA512]::
{{range $line := (multiline $sha512 32)}}
   {{$line}}{{end}}

{{end}}

