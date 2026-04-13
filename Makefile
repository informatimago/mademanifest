MMDC ?= mmdc
MERMAID_SRC := architecture/diagram.mmd
MERMAID_PNG := architecture/diagram.png

.PHONY: all diagram clean

all: diagram

diagram: $(MERMAID_PNG)

$(MERMAID_PNG): $(MERMAID_SRC)
	$(MMDC) -i $(MERMAID_SRC) -o $(MERMAID_PNG)

clean:
	rm -f $(MERMAID_PNG)
