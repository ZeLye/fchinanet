echo "start build in linux";
CGO_ENABLED=0 GOOS=linux  go build -o build/fchinanet_linux/fchinanet fchinanet.go;
echo "complete build in linux";
echo "start build in windows";
CGO_ENABLED=0 GOOS=windows go build -o build/fchinanet_windows/fchinanet.exe fchinanet.go;
echo "complete build in windows";
echo "start build in macos";
CGO_ENABLED=0 GOOS=darwin go build -o build/fchinanet_mac/fchinanet fchinanet.go;
echo "complete build in macos";