# GMASH

Give Me A SHell

Allow others to temporarily and securely ssh into your account.

# Install
If you have go installed

`> go get github.com/efarrer/gmash`

# Usage
Allow someone to securely login to your local account.

`> ./gmash`

Only allow connections from your local network

`> ./gmash -local`

If you want to share your session with another user you can do so with tmux or screen.

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
