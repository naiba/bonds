# Structured ORG compatibility patch

Base: github.com/emersion/go-vcard v0.1.0 (MIT; original LICENSE retained).
All original production and test files are included. Only card.go, decoder.go,
encoder.go and the added organization.go/organization_test.go differ from that base.

The original decoder unescapes backslashes before a caller can distinguish a
literal semicolon from a component separator. The encoder also lacks component
escaping. Application-only splitting cannot recover that lost information.

This patch preserves decoded ORG components in private Field metadata while
keeping Value's existing text representation. Field.Organization and
NewOrganization expose unescaped components; the encoder escapes each component
individually. Other properties and codecs remain unchanged. No protocol-visible
private marker or runtime monkey patch is used.

Remove this local replacement when upstream provides equivalent component-aware
ORG decoding and encoding. Keep the organization codec, import/subscription and
HTTP regression tests when upgrading. Run this module's upstream and added tests
with `go test ./...`, as well as the application's dual-database vCard/DAV suites.
