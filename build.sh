# @Author: 01sr
# @Date:   2018-04-05 09:50:43
# @Last Modified by:   01sr
# @Last Modified time: 2018-04-05 10:01:05
echo "start build, linux_arm";
CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -o fchinanet_linux_arm Chinanet.go;
echo "complete build, linux_arm";
echo "start build, linux_amd64";
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o fchinanet_linux_amd64 Chinanet.go;
echo "complete build, linux_mips";
echo "start build, linux_mips";
CGO_ENABLED=0 GOOS=linux GOARCH=mips go build -o fchinanet_linux_mips Chinanet.go;
echo "complete build, linux_mips32";
echo "start build, linux_mips32le";
CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -o fchinanet_linux_mips32le Chinanet.go;
echo "complete build, linux_mips32le";