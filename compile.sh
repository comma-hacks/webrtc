#!/bin/bash
set -euo pipefail

# Cap’n Proto 
pushd cereal
git checkout *.capnp
rm -rf gen/go
for f in *.capnp; do
  name=$(echo $f | sed 's/.capnp//')
  mv $f $f.bak
  echo "using Go = import \"/go.capnp\";" > $name.capnp
  echo "\$Go.package(\"cereal\");" >> $name.capnp
  echo "\$Go.import(\"cereal\");" >> $name.capnp
  cat $f.bak >> $name.capnp
  rm $f.bak
done
mkdir gen/go
go install capnproto.org/go/capnp/v3/capnpc-go@latest
capnp compile -I$(go env GOPATH)/src/capnproto.org/go/capnp/std -o go:gen/go *.capnp
git checkout *.capnp
cat <<EOF > gen/go/go.mod
module github.com/commaai/cereal

go 1.19
EOF
popd


# Go program
go build -o webrtc-body main.go