module github.com/sequix/dequash-snapshotter

go 1.14

require (
	github.com/VictoriaMetrics/fastcache v1.5.7
	github.com/containerd/containerd v1.3.0
	github.com/containerd/continuity v0.0.0-20200710164510-efbc4488d8fe
	github.com/containerd/ttrpc v1.0.1 // indirect
	github.com/containerd/typeurl v1.0.1 // indirect
	github.com/datadog/zstd v1.4.5 // indirect
	github.com/dchest/siphash v1.2.1 // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/folbricht/desync v0.8.0
	github.com/go-ini/ini v1.57.0 // indirect
	github.com/google/go-containerregistry v0.1.1
	github.com/google/uuid v1.1.1
	github.com/minio/minio-go v6.0.14+incompatible
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pkg/sftp v1.11.0 // indirect
	github.com/pkg/xattr v0.4.1 // indirect
	github.com/sanity-io/litter v1.2.0
	github.com/stretchr/testify v1.6.1 // indirect
	go.uber.org/zap v1.15.0
	golang.org/x/net v0.0.0-20200625001655-4c5254603344 // indirect
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208 // indirect
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/grpc v1.30.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

replace github.com/containerd/containerd => github.com/containerd/containerd v1.3.0
