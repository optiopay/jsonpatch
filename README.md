# JSON Patch

This module provides an implementation of [RFC 6902](http://tools.ietf.org/html/rfc6902). 

There are other go libraries that provide similar functionality. The difference between the rest and this is that instead of using the patch to create a JSON []byte array it applies the patch to a go type.

The library exposes two APIs `Apply` and `Diff`(to be done). 

    func Apply(data []byte, x interface{}) error

    func Diff(a, b interface{}) ([]byte, error)

It should be noted that `Apply` makes a recursive copy of the value passed to the function. It applies the changes only if all of the operations in the patch succeeded.


The repository also provides a module `deep` which exposes an API `Copy`.

    func Copy(x, y interface{}) error

It makes a recursive copy of x into y.