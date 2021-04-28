TBBS Test Report
================

.. toctree::
   :maxdepth: 2
   :caption: Contents:

Bagits
======

.. csv-table:: a title
   :header: "Name", "Size", "Ingest Date", "Tests"

   {{range . }}
   "{{.Name}}", {{.Size}}, "{{.Ingested}}", "{{range .Tests}}{{.Location}} ({{.Status}}, {{.Date}} // {{end}}"
   {{end}}

Indices and tables
==================

* :ref:`genindex`
* :ref:`modindex`
* :ref:`search`
