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
  cat $f |\
  # Remove the C++ imports
  grep -v 'c++.capnp' | grep -v '$Cxx.namespace' |\
  # Avoid collisions by annotating
  sed 's/message @6 :Text;/message @6 :Text $Go.name("text");/'\
  >> $out
done
# go install capnproto.org/go/capnp/v3/capnpc-go@latest
# capnp compile -I$(go env GOPATH)/src/capnproto.org/go/capnp/std -o go:../../gen/go *.capnp
capnp compile -I../../../go-capnproto2/std -o go:../../gen/go *.capnp
popd # leave ./tmp/go
rm -rf tmp

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
export CGO_LDFLAGS="-L$PWD/go-astiav/tmp/n4.4.1/lib/"
export CGO_CXXFLAGS="-I$PWD/go-astiav/tmp/n4.4.1/include/"
export PKG_CONFIG_PATH="$PWD/go-astiav/tmp/n4.4.1/lib/pkgconfig"
mkdir -p dist
go build -o dist/go-webrtc-body .