{{.Name}}
{{repeat "=" (len .Name)}}

Checksums
---------

{{range $cstype, $cs := .Checksums}}
{{$cstype}}::
{{range $line := (multiline $cs 64)}}
   {{$line}}{{end}}

{{end}}

{{if gt (len .Indexer.Errors) 0}}
Errors
------

{{range $key, $val := .Indexer.Errors}}{{$key}}
| {{quote $val}}
{{end}}
{{end}}
{{if gt (len .Indexer.NSRL) 0}}
NSRL
----
{{range $k, $nsrl := .Indexer.NSRL}}
.. list-table:: NSRL #{{$k}}

{{range $key, $val := $nsrl.File}}   * - File
     - {{$key}}
     - {{$val}}
{{end}}{{range $key, $val := $nsrl.FileMfG}}   * - FileMfg
     - {{$key}}
     - {{$val}}
{{end}}{{range $key, $val := $nsrl.OS}}   * - OS
     - {{$key}}
     - {{$val}}
{{end}}{{range $key, $val := $nsrl.OSMfg}}   * - OSMfg
     - {{$key}}
     - {{$val}}
{{end}}{{range $key, $val := $nsrl.Prod}}   * - Prod
     - {{$key}}
     - {{$val}}
{{end}}{{range $key, $val := $nsrl.ProdMfg}}   * - ProdMfg
     - {{$key}}
     - {{$val}}
{{end}}{{end}}
{{end}}
{{if gt (len .Indexer.Siegfried) 0}}
Siegfried
---------
{{range $idx := .Indexer.Siegfried}}
| Pronom ID: {{$idx.ID}}
| Name: {{$idx.Name}}
| Mimetype: {{$idx.MIME}}
{{end}}
{{end}}
{{if gt (len .Indexer.Identify) 0}}
ImageMagick
-----------

.. list-table:: Identify

{{range $key, $val := .Indexer.Identify.image}}   * - {{$key}}
     - {{$val}}
{{end}}
{{end}}
{{if gt (len .Indexer.Clamav) 0}}
Clamav
------

.. list-table:: Clamav

{{range $key, $val := .Indexer.Clamav}}   * - {{$key}}
     - {{quote $val}}
{{end}}
{{end}}
{{if not (eq .Indexer.FFProbe.Format.FormatName "")}}
FFProbe
-------

{{if gt (len .Indexer.FFProbe.Format.Tags) 0}}
.. list-table:: Metadata

{{range $key, $val := .Indexer.FFProbe.Format.Tags}}   * - {{$key}}
     - {{quote $val}}
{{end}}
{{end}}

.. list-table:: Format

   * - FormatName
     - {{.Indexer.FFProbe.Format.FormatName}}
   * - FormatLongName
     - {{.Indexer.FFProbe.Format.FormatLongName}}
   * - NbStreams
     - {{.Indexer.FFProbe.Format.NbStreams}}
   * - NbPrograms
     - {{.Indexer.FFProbe.Format.NbPrograms}}
   * - Duration
     - {{.Indexer.FFProbe.Format.Duration}}
   * - Size
     - {{.Indexer.FFProbe.Format.Size}}
   * - BitRate
     - {{.Indexer.FFProbe.Format.BitRate}}
   * - ProbeScore
     - {{.Indexer.FFProbe.Format.ProbeScore}}

{{range $key, $stream := .Indexer.FFProbe.Streams}}
.. list-table:: Stream #{{$key}}

   * - Codec
     - {{$stream.CodecName}} (({{$stream.CodecLongName}})
   {{if ne $stream.Profile ""}}* - Profile
     - {{$stream.Profile}}
   {{end}}* - CodecType
     - {{$stream.CodecType}}
   {{if ne $stream.CodecTimeBase ""}}* - CodecTimeBase
     - {{$stream.CodecTimeBase}}
   {{end}}{{if ne $stream.CodecTagString ""}}* - CodecTag
     - {{$stream.CodecTagString}} ({{$stream.CodecTag}})
   {{end}}{{if gt $stream.Width 0}}* - Width
     - {{$stream.Width}}
   * - Height
     - {{$stream.Height}}
   {{end}}{{if gt $stream.CodedWidth 0}}* - CodedWidth
     - {{$stream.CodedWidth}}
   * - CodedHeight
     - {{$stream.CodedHeight}}
   {{end}}* - Has B-Frames
     - {{$stream.HasBFrames}}
   {{if ne $stream.SampleAspectRatio ""}}* - Aspect Ratio
     - Sample: {{$stream.SampleAspectRatio}} // Display: {{$stream.DisplayAspectRatio}}
   {{end}}{{if ne $stream.PixFmt ""}}* - PixFmt
     - {{$stream.PixFmt}}
   {{end}}* - Level
     - {{$stream.Level}}
   {{if ne $stream.ChromaLocation ""}}* - ChromaLocation
     - {{$stream.ChromaLocation}}
   {{end}}{{if gt $stream.Refs 0}}* - Refs
     - {{$stream.Refs}}
   {{end}}{{if ne $stream.QuarterSample ""}}* - QuarterSample
     - {{$stream.QuarterSample}}
   {{end}}{{if ne $stream.DivxPacked ""}}* - DivxPacked
     - {{$stream.DivxPacked}}
   {{end}}* - Frame rate
     - R: {{$stream.RFrameRrate}} // Avg: {{$stream.AvgFrameRate}}
   * - Time base
     - {{$stream.TimeBase}}
   * - Duration
     - {{$stream.Duration}} ({{$stream.DurationTs}})
   * - Disposition
     - {{$stream.Disposition}}
   * - BitRate
     - {{$stream.BitRate}}

{{end}}
{{end}}
{{if gt (len .Indexer.Exif) 0}}
Exif
----

.. list-table::

{{range $key, $val := .Indexer.Exif}}   * - {{$key}}
     - {{$val}}
{{end}}
{{end}}
{{if gt (len .Indexer.Tika) 0}}
Tika
----

{{range $key, $val := (index .Indexer.Tika 0)}}:{{$key}}:
{{range $key,$line := (linebreak (quote $val) 60)}}  {{$line}}
{{end}}{{end}}
{{end}}
