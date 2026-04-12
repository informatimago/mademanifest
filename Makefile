.PHONY: all pdf clean

EMACS ?= emacs
ORG_FILES := $(wildcard specifications/*.org)
PDF_FILES := $(ORG_FILES:.org=.pdf)
LATEX_AUX_EXTS := aux log out toc tex fls fdb_latexmk

all: pdf

pdf: $(PDF_FILES)

specifications/%.pdf: specifications/%.org
	$(EMACS) --batch "$<" \
		--eval "(require 'ox-latex)" \
		--eval "(org-latex-export-to-pdf)"

clean:
	rm -f $(PDF_FILES)
	rm -f $(foreach ext,$(LATEX_AUX_EXTS),$(ORG_FILES:.org=.$(ext)))
