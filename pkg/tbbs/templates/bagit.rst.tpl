{{repeat "*" (len .Bagit.Name)}}
{{.Bagit.Name}}
{{repeat "*" (len .Bagit.Name)}}

.. toctree::

   contents.rst
{{range $file := .Files}}   {{$file}}
{{end}}
