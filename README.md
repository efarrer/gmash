# gmash
A simple way to open a secure ssh shell to your account.

# Getting dependencies 
go get -u github.com/golang/dep/... && dep ensure

# Testing
go test $(go list ./... | grep -v /vendor/)

# Building
go build

# Running
./gmash
