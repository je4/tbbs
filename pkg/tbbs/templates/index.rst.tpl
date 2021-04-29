TBBS Test Report
================

.. toctree::
   :maxdepth: 2
   :numbered:
   :titlesonly:
   :glob:
   :hidden:


{{range . }}
{{.Name}}.rst{{end}}

report generated at {{now}}.

Bagits
======

.. csv-table::
   :header: "Name", "Size", "Ingest Date", "Tests"

   {{range . }}"{{.Name}}", {{.Size}}, "{{.Ingested}}", "{{.TestsMessage}}"
   {{end}}

Indices and tables
==================

* :ref:`genindex`
* :ref:`modindex`
* :ref:`search`
