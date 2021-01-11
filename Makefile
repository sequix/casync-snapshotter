OUTDIR = out
.PHONY: all convert mount snapshotter clean

all: convert mount snapshotter

convert:
	go build -o $(OUTDIR)/convert cmd/convert/main.go

mount:
	go build -o $(OUTDIR)/mount cmd/mount/main.go

snapshotter:
	go build -o $(OUTDIR)/snapshotter cmd/snapshotter/main.go

clean:
	rm -rf $(OUTDIR)
