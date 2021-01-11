package main

import (
	"flag"

	"github.com/sequix/casync-snapshotter/pkg/buildinfo"
	"github.com/sequix/casync-snapshotter/pkg/log"
)

// TODO: filesystem cache 单独写一个 go 库；
// TODO：所有的 store 实现都在一个 store 目录下，还是通过flag来控制怎么生存 store 对象，保持一致性 （毕竟，store在一个进程中只需要一个）

func main() {
	flag.Parse()
	buildinfo.Init()
	log.Init("casnap-cnksvr")

	// 1.server with proper handler
	// 2.memcache -> fscache -> p2pcache -> s3
}