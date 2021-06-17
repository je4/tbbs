*********
Überblick
*********

Inhalt
======

.. code-block::
   :caption: Baginfo

{{range $key, $line := (string2array .Bagit.Baginfo)}}   {{$line}}
{{end}}

.. tabularcolumns:: |l|l|l|r|

.. list-table:: Content
   :header-rows: 1

{{.Contents.DrawTableList}}

Integritätstests
================

.. tabularcolumns::
   |m{0.10\linewidth}|m{0.20\linewidth}|m{0.10\linewidth}|m{0.60\linewidth}|


{{range $loc, $table := .Tests}}
.. tabularcolumns::
   |m{0.10\textwidth}|m{0.20\textwidth}|m{0.10\textwidth}|m{0.60\textwidth}|

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

