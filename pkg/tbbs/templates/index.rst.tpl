TBBS Test Report
================

{{ .BagitTable.DrawTable }}

.. list-table:: Bagits
   :widths: 25 20 20 35
   :header-rows: 1

   * - Name
     - Size
     - Ingest Date
     - Tests
   {{range .Bagits }}* - {{.Name}}
     - ..right:: {{.Size}}
     - {{.Ingested}}
     - {{.TestsMessage}}
   {{end}}


report generated at {{now}}.

