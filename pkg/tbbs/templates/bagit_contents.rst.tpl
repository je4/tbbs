*********
Überblick
*********

Inhalt
======
.. tabularcolumns:: |l|l|l|r|

.. list-table:: Content
   :header-rows: 1

{{.Contents.DrawTableList}}

Integritätstests
================

{{range $loc, $table := .Tests}}
.. list-table:: {{$loc}}

{{$table.DrawTableList}}
{{end}}

Transfer (File to Storage / Time)
=================================

{{range $loc, $transfer := .Transfer}}| Transfer to "{{$loc}}" from {{$transfer.Start}} to {{$transfer.End}}: {{$transfer.Status}}
{{end}}

SHA512 Prüfsummen Übersicht
===========================
{{range $file, $sha512 := .SHA512}}
{{$file}} [SHA512]::
{{range $line := (multiline $sha512 64)}}
   {{$line}}{{end}}

{{end}}

