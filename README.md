# jtv2xmltv
Converts jtv file to xml. Written in Go. Converts 1Mb jtv file in 0.5s compared to python's 3.5s

jtv2xmltv in repository build with go1.4 for CentOS 5

# Usage
```
jtv2xmltv file.zip > output.xml
```

# Compilation for CentOS 5
install https://github.com/moovweb/gvm and its dependencies
```
gvm install go1.4
gvm use go1.4 --default
go get golang.org/x/text/encoding/charmap
go build jtv2xmltv.go
```
