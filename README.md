# gmash
A simple way to open a secure ssh shell to your account.

# Getting dependencies 
go get -u github.com/golang/dep/... && dep ensure

# Running unit tests
go test $(go list ./... | grep -v /vendor/)

# Testing go get install (with docker)
docker run --rm -it golang go get github.com/efarrer/gmash

# Building
go build

# Running
./gmash
