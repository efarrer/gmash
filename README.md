# GMASH

Give Me A SHell

Allow others temporarily and securely ssh into your account.

# Usage
Allow someone on your local network to login to your account

`> ./gmash -local`

Allow someone not on your local network to login to your account. You must have ngrok installed and configured

`> ./gmash -global`

# Development

## Building
1. Download third party dependencies

`> go get -u github.com/golang/dep/... && dep ensure`

2. Compile

`> go build`

## Testing
1. Running unit tests

`> go test $(go list ./... | grep -v /vendor/)`

2. Ensure code pushed is go get'able

`> docker run --rm -it golang go get github.com/efarrer/gmash`
