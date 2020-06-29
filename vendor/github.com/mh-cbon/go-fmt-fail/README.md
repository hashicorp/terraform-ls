# go-fmt-fail

`go fmt` fails when a file is formatted.

This tool is part of the [go-github-release workflow](https://github.com/mh-cbon/go-github-release)

## Install

Pick an msi package [here](https://github.com/mh-cbon/go-fmt-fail/releases)!

__deb/ubuntu/rpm repositories__

```sh
wget -O - https://raw.githubusercontent.com/mh-cbon/latest/master/source.sh \
| GH=mh-cbon/go-fmt-fail sh -xe
# or
curl -L https://raw.githubusercontent.com/mh-cbon/latest/master/source.sh \
| GH=mh-cbon/go-fmt-fail sh -xe
```

__deb/ubuntu/rpm packages__

```sh
curl -L https://raw.githubusercontent.com/mh-cbon/latest/master/install.sh \
| GH=mh-cbon/go-fmt-fail sh -xe
# or
wget -q -O - --no-check-certificate \
https://raw.githubusercontent.com/mh-cbon/latest/master/install.sh \
| GH=mh-cbon/go-fmt-fail sh -xe
```

__chocolatey__

```sh
choco install go-fmt-fail -y
```

__go__

```sh
mkdir -p $GOPATH/src/github.com/mh-cbon
cd $GOPATH/src/github.com/mh-cbon
git clone https://github.com/mh-cbon/go-fmt-fail.git
cd go-fmt-fail
glide install
go install
```

# Usage

```sh
go-fmt-fail [packages or files]
```
