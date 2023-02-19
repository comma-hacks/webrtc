#!/bin/bash
set -euxo pipefail

# Cap’n Proto 
pushd cereal
rm -rf gen/go tmp/go
mkdir -p gen/go tmp/go
pushd tmp/go
for f in ../../*.capnp; do
  out=$(basename $f)
  echo "using Go = import \"/go.capnp\";" > $out
  echo "\$Go.package(\"cereal\");" >> $out
  echo "\$Go.import(\"cereal\");" >> $out
  cat $f | grep -v 'c++.capnp' | grep -v '$Cxx.namespace' >> $out
done
go install capnproto.org/go/capnp/v3/capnpc-go@latest
capnp compile -I$(go env GOPATH)/src/capnproto.org/go/capnp/std -o go:../../gen/go *.capnp
popd # leave ./tmp/go

## Services

cat <<EOF > gen/go/services.go
// Code generated. DO NOT EDIT.

package cereal

type Service struct {
	Name       string
	Port       int
	ShouldLog bool
	Frequency  int
	Decimation int
}

var services = []Service{
EOF
grep '  {' services.h >> gen/go/services.go
cat <<EOF >> gen/go/services.go
}

func GetServices() []Service {
  return services
}
EOF

cat <<EOF > gen/go/go.mod
// Code generated. DO NOT EDIT.

module github.com/commaai/cereal

go 1.19
EOF
popd # leave ./cereal

# Go program
# go build -o webrtc-body main.go