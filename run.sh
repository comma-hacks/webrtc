export CGO_LDFLAGS="-L$PWD/go-astiav/tmp/n4.4.1/lib/"
export CGO_CXXFLAGS="-I$PWD/go-astiav/tmp/n4.4.1/include/"
export PKG_CONFIG_PATH="$PWD/go-astiav/tmp/n4.4.1/lib/pkgconfig"
go run .