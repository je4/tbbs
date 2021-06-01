package tbbs

type IngestBagitContent struct {
	bagit                   *IngestBagit
	contentId               int64
	ZipPath, DiskPath       string
	Filesize                int64
	Checksums               map[string]string
	Mimetype                string
	Width, Height, Duration int64
	Indexer                 string
}

func (ibc *IngestBagitContent) Store() error {
	_, err := ibc.bagit.ingest.bagitContentStore(ibc)
	return err
}
