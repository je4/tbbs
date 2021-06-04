TOPTARGETS := all

SUBDIRS := $(wildcard */.)

$(TOPTARGETS): $(SUBDIRS)
$(SUBDIRS):
	$(MAKE) -C $@ latexpdf
	cp -f $@/_build/latex/*.pdf .
	$(MAKE) -C $@ clean

.PHONY: $(TOPTARGETS) $(SUBDIRS)