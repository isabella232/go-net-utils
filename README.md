# Go Net Utilities

## Summary

Currently the purpose of this repo is to provide mechanisms for easily tracking bytes transferred at the connection level.

Typical usage involves using the track package's version of a Go net or net/http interface/type in place of the standard library's version. In cases where an interface is not present, there will be a struct providing tracking functionality with access to the underlying standard library type.
