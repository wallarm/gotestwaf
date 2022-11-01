# go-emailaddress #

[![GoDoc](https://godoc.org/github.com/mcnijman/go-emailaddress?status.svg)](https://godoc.org/github.com/mcnijman/go-emailaddress) [![Build Status](https://travis-ci.org/mcnijman/go-emailaddress.svg?branch=master)](https://travis-ci.org/mcnijman/go-emailaddress) [![Test Coverage](https://coveralls.io/repos/github/mcnijman/go-emailaddress/badge.svg?branch=master)](https://coveralls.io/github/mcnijman/go-emailaddress?branch=master) [![go report](https://goreportcard.com/badge/github.com/mcnijman/go-emailaddress)](https://goreportcard.com/report/github.com/mcnijman/go-emailaddress)

go-emailaddress is a tiny Go library for finding, parsing and validating email addresses. This
library is tested for Go v1.9 and above.

Note that there is no such thing as perfect email address validation other than sending an actual
email (ie. with a confirmation token). This library however checks if the email format conforms to
the spec and if the host (domain) is actually able to receive emails. You can also use this library
to find emails in a byte array. This package was created as similar packages don't seem to be
maintained anymore (ie contain bugs with pull requests still open), and/or use wrong local
validation.

## Usage ##

```bash
go get -u github.com/mcnijman/go-emailaddress
```

### Parsing and local validation ###

Parse and validate the email locally using RFC 5322 regex, note that when `err == nil` it doesn't
necessarily mean the email address actually exists.

```go
import "github.com/mcnijman/go-emailaddress"

email, err := emailaddress.Parse("foo@bar.com")
if err != nil {
    fmt.Println("invalid email")
}

fmt.Println(email.LocalPart) // foo
fmt.Println(email.Domain) // bar.com
fmt.Println(email) // foo@bar.com
fmt.Println(email.String()) // foo@bar.com
```

### Validating the host ###

Host validation will first attempt to resolve the domain and then verify if we can start a mail
transaction with the host. This is relatively slow as it will contact the host several times.
Note that when `err == nil` it doesn't necessarily mean the email address actually exists.

```go
import "github.com/mcnijman/go-emailaddress"

email, err := emailaddress.Parse("foo@bar.com")
if err != nil {
    fmt.Println("invalid email")
}

err := email.ValidateHost()
if err != nil {
    fmt.Println("invalid host")
}
```

### Finding emails ###

This will look for emails in a byte array (ie text or an html response).

```go
import "github.com/mcnijman/go-emailaddress"

text := []byte(`Send me an email at foo@bar.com.`)
validateHost := false

emails := emailaddress.Find(text, validateHost)

for _, e := range emails {
    fmt.Println(e)
}
// foo@bar.com
```

As RFC 5322 is really broad this method will likely match images and urls that contain
the '@' character (ie. !--logo@2x.png). For more reliable results, you can use the following method.

```go
import "github.com/mcnijman/go-emailaddress"

text := []byte(`Send me an email at foo@bar.com or fake@domain.foobar.`)
validateHost := false

emails := emailaddress.FindWithIcannSuffix(text, validateHost)

for _, e := range emails {
    fmt.Println(e)
}
// foo@bar.com
```

## Versioning ##

This library uses [semantic versioning 2.0.0](https://semver.org/spec/v2.0.0.html).

## License ##

This library is distributed under the MIT license found in the [LICENSE](./LICENSE)
file.