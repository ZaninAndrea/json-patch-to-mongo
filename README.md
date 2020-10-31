# JSON Patches to Mongo

This utility converts [JSON patches](http://jsonpatch.com/) to MongoDB update queries.

## Using the package

To install the package run

```
go get github.com/ZaninAndrea/jsonpatch-to-mongo
```

Then you can use the utility simply by passing the JSON Patches as `[]byte` to the `ParsePatches` function

```go
patches := []byte(`[
    { "op": "add", "path": "/hello/0/hi/5", "value": 1 },
    { "op": "add", "path": "/hello/0/hi/5", "value": 2 },
    { "op": "replace", "path": "/world", "value": {"foo":"bar"} }
]`)
updateQuery, err := ParsePatches(patches)
```

## Credits

This is a Go port of the JavaScript library [jsonpatch-to-mongodb](https://github.com/mongodb-js/jsonpatch-to-mongodb).
